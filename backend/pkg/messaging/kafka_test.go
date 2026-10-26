package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"
)

// fakeGroup ทำตาม sarama.ConsumerGroup เฉพาะ Consume (method อื่นไม่ได้ใช้ใน test นี้)
type fakeGroup struct {
	sarama.ConsumerGroup
	results []error        // ค่าที่ Consume จะคืนในแต่ละครั้ง
	onCall  func(call int) // hook ให้ test สั่ง cancel ได้ตอนเรียกครั้งที่ n
	calls   int
}

func (f *fakeGroup) Consume(context.Context, []string, sarama.ConsumerGroupHandler) error {
	f.calls++
	if f.onCall != nil {
		f.onCall(f.calls)
	}
	if f.calls <= len(f.results) {
		return f.results[f.calls-1]
	}
	return nil
}

func recordSleep(waits *[]time.Duration) func(context.Context, time.Duration) error {
	return func(ctx context.Context, d time.Duration) error {
		*waits = append(*waits, d)
		return ctx.Err()
	}
}

func TestRunConsumerGroup_BacksOffOnErrorsAndResetsAfterSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	brokerDown := errors.New("kafka: client has run out of available brokers")
	g := &fakeGroup{
		// พัง 3 ครั้ง → สำเร็จ 1 ครั้ง (รีเซ็ต) → พังอีก 1 ครั้ง → ครั้งที่ 6 สั่ง shutdown
		results: []error{brokerDown, brokerDown, brokerDown, nil, brokerDown},
		onCall: func(call int) {
			if call == 6 {
				cancel()
			}
		},
	}
	var waits []time.Duration

	runConsumerGroup(ctx, "test", g, []string{"t"}, nil, recordSleep(&waits))

	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, time.Second}
	if len(waits) != len(want) {
		t.Fatalf("waits = %v, want %v", waits, want)
	}
	for i := range want {
		if waits[i] != want[i] {
			t.Errorf("wait[%d] = %s, want %s (all waits %v)", i, waits[i], want[i], waits)
		}
	}
	if g.calls != 6 {
		t.Errorf("Consume calls = %d, want 6", g.calls)
	}
}

func TestRunConsumerGroup_CapsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	down := errors.New("down")
	g := &fakeGroup{results: make([]error, 10), onCall: func(call int) {
		if call == 10 {
			cancel()
		}
	}}
	for i := range g.results {
		g.results[i] = down
	}
	var waits []time.Duration

	runConsumerGroup(ctx, "test", g, nil, nil, recordSleep(&waits))

	if last := waits[len(waits)-1]; last != consumeRetryMax {
		t.Errorf("last wait = %s, want capped at %s (all waits %v)", last, consumeRetryMax, waits)
	}
}

func TestRunConsumerGroup_StopsWhenGroupClosed(t *testing.T) {
	g := &fakeGroup{results: []error{sarama.ErrClosedConsumerGroup}}
	var waits []time.Duration

	runConsumerGroup(context.Background(), "test", g, nil, nil, recordSleep(&waits))

	if g.calls != 1 || len(waits) != 0 {
		t.Errorf("calls = %d waits = %v, want 1 call and no retry after group closed", g.calls, waits)
	}
}

func TestRunConsumerGroup_ShutdownDuringBackoffExitsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	g := &fakeGroup{results: []error{errors.New("down"), errors.New("down")}}
	sleep := func(context.Context, time.Duration) error {
		cancel() // shutdown เกิดขึ้นระหว่างรอ backoff
		return ctx.Err()
	}

	runConsumerGroup(ctx, "test", g, nil, nil, sleep)

	if g.calls != 1 {
		t.Errorf("Consume calls = %d, want 1 (must not retry after shutdown)", g.calls)
	}
}
