package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"
)

type fakeHandler struct {
	errs  []error // error ที่จะคืนในแต่ละครั้ง (ครั้งที่เกินจำนวนคืน nil)
	calls int
}

func (f *fakeHandler) Handle(context.Context, *sarama.ConsumerMessage) error {
	f.calls++
	if f.calls <= len(f.errs) {
		return f.errs[f.calls-1]
	}
	return nil
}

type fakeDLQ struct {
	err      error
	calls    int
	cause    error
	attempts int
}

func (f *fakeDLQ) Publish(_ context.Context, _ string, _ *sarama.ConsumerMessage, cause error, attempts int) error {
	f.calls++
	f.cause = cause
	f.attempts = attempts
	return f.err
}

func newTestHandler(h Handler, dlq DeadLetterPublisher) (*ConsumerGroupHandler, *[]time.Duration) {
	waits := []time.Duration{}
	c := NewConsumerGroupHandler("test", h, dlq, DefaultRetryPolicy())
	c.sleep = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}
	return c, &waits
}

func testMessage() *sarama.ConsumerMessage {
	return &sarama.ConsumerMessage{Topic: "order.events", Partition: 2, Offset: 41, Key: []byte("order-1"), Value: []byte(`{}`)}
}

func TestProcess_SucceedsAfterTransientErrors(t *testing.T) {
	transient := errors.New("db timeout")
	h := &fakeHandler{errs: []error{transient, transient}}
	dlq := &fakeDLQ{}
	c, waits := newTestHandler(h, dlq)

	if err := c.process(context.Background(), testMessage()); err != nil {
		t.Fatalf("process() = %v, want nil", err)
	}
	if h.calls != 3 {
		t.Errorf("handler calls = %d, want 3", h.calls)
	}
	if dlq.calls != 0 {
		t.Errorf("dlq calls = %d, want 0 (message eventually succeeded)", dlq.calls)
	}
	want := []time.Duration{500 * time.Millisecond, time.Second}
	if len(*waits) != len(want) || (*waits)[0] != want[0] || (*waits)[1] != want[1] {
		t.Errorf("backoff waits = %v, want %v", *waits, want)
	}
}

func TestProcess_SendsToDLQAfterMaxAttempts(t *testing.T) {
	transient := errors.New("db timeout")
	h := &fakeHandler{errs: []error{transient, transient, transient, transient, transient}}
	dlq := &fakeDLQ{}
	c, _ := newTestHandler(h, dlq)

	if err := c.process(context.Background(), testMessage()); err != nil {
		t.Fatalf("process() = %v, want nil (message parked in DLQ)", err)
	}
	if h.calls != 4 {
		t.Errorf("handler calls = %d, want 4", h.calls)
	}
	if dlq.calls != 1 || dlq.attempts != 4 || !errors.Is(dlq.cause, transient) {
		t.Errorf("dlq got calls=%d attempts=%d cause=%v, want 1, 4, %v", dlq.calls, dlq.attempts, dlq.cause, transient)
	}
}

func TestProcess_PermanentErrorSkipsRetry(t *testing.T) {
	h := &fakeHandler{errs: []error{Permanent(errors.New("bad json"))}}
	dlq := &fakeDLQ{}
	c, waits := newTestHandler(h, dlq)

	if err := c.process(context.Background(), testMessage()); err != nil {
		t.Fatalf("process() = %v, want nil", err)
	}
	if h.calls != 1 || len(*waits) != 0 {
		t.Errorf("handler calls = %d waits = %v, want 1 call and no waits", h.calls, *waits)
	}
	if dlq.calls != 1 || dlq.attempts != 1 {
		t.Errorf("dlq calls = %d attempts = %d, want 1, 1", dlq.calls, dlq.attempts)
	}
}

func TestProcess_DLQFailureKeepsMessageUncommitted(t *testing.T) {
	h := &fakeHandler{errs: []error{Permanent(errors.New("bad json"))}}
	dlq := &fakeDLQ{err: errors.New("broker unavailable")}
	c, _ := newTestHandler(h, dlq)

	if err := c.process(context.Background(), testMessage()); err == nil {
		t.Fatal("process() = nil, want error so the offset is not marked")
	}
}

func TestProcess_CancelledContextIsNotDeadLettered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := &fakeHandler{errs: []error{errors.New("context canceled mid-query")}}
	dlq := &fakeDLQ{}
	c, _ := newTestHandler(h, dlq)
	cancel()

	if err := c.process(ctx, testMessage()); err == nil {
		t.Fatal("process() = nil, want ctx error")
	}
	if dlq.calls != 0 {
		t.Errorf("dlq calls = %d, want 0 (rebalance is not the message's fault)", dlq.calls)
	}
}

// blockingHandler จำลอง DB ที่ไม่ตอบ: ค้างจนกว่า ctx จะถูกตัด
type blockingHandler struct{ calls int }

func (b *blockingHandler) Handle(ctx context.Context, _ *sarama.ConsumerMessage) error {
	b.calls++
	<-ctx.Done()
	return ctx.Err()
}

func TestProcess_AttemptTimeoutRetriesThenDeadLetters(t *testing.T) {
	h := &blockingHandler{}
	dlq := &fakeDLQ{}
	c, _ := newTestHandler(h, dlq)
	c.policy.AttemptTimeout = 20 * time.Millisecond

	start := time.Now()
	if err := c.process(context.Background(), testMessage()); err != nil {
		t.Fatalf("process() = %v, want nil (message parked in DLQ)", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("process took %s, want each attempt cut at ~20ms", elapsed)
	}
	if h.calls != 4 {
		t.Errorf("handler calls = %d, want 4 (timeout is transient, must retry)", h.calls)
	}
	if dlq.calls != 1 || !errors.Is(dlq.cause, context.DeadlineExceeded) {
		t.Errorf("dlq calls = %d cause = %v, want 1 and DeadlineExceeded", dlq.calls, dlq.cause)
	}
}

func TestBackoff_CapsAtMax(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 10, InitialBackoff: time.Second, MaxBackoff: 3 * time.Second}
	got := []time.Duration{p.backoff(1), p.backoff(2), p.backoff(3), p.backoff(8)}
	want := []time.Duration{time.Second, 2 * time.Second, 3 * time.Second, 3 * time.Second}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("backoff(%d) = %s, want %s", i+1, got[i], want[i])
		}
	}
}

func TestBuildDeadLetterMessage_KeepsOriginalHeadersAndKey(t *testing.T) {
	msg := testMessage()
	msg.Headers = []*sarama.RecordHeader{{Key: []byte("EventType"), Value: []byte("ORDER_CREATED")}}

	out := buildDeadLetterMessage("product", msg, errors.New("boom"), 4, time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC))

	if out.Topic != "order.events.dlq" {
		t.Errorf("topic = %s, want order.events.dlq", out.Topic)
	}
	if out.Key == nil {
		t.Error("key dropped, want original key kept for partition ordering on replay")
	}
	got := map[string]string{}
	for _, h := range out.Headers {
		got[string(h.Key)] = string(h.Value)
	}
	for k, v := range map[string]string{
		"EventType":             "ORDER_CREATED",
		HeaderOriginalTopic:     "order.events",
		HeaderOriginalPartition: "2",
		HeaderOriginalOffset:    "41",
		HeaderConsumer:          "product",
		HeaderError:             "boom",
		HeaderAttempts:          "4",
		HeaderFailedAt:          "2026-10-19T00:00:00Z",
	} {
		if got[k] != v {
			t.Errorf("header %s = %q, want %q", k, got[k], v)
		}
	}

	msg.Key = nil
	if out := buildDeadLetterMessage("product", msg, errors.New("boom"), 1, time.Now()); out.Key != nil {
		t.Error("nil key became non-nil encoder, would pin every DLQ message to one partition")
	}
}
