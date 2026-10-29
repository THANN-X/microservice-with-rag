package worker

import (
	"context"
	"errors"
	"order_service/internal/core/domain"
	"testing"
	"time"

	"github.com/IBM/sarama"
)

type fakeOutboxRepo struct {
	pending     []*domain.OutboxEvent
	sent        []string
	incremented []string
	failed      []string
	fetchErr    error
}

func (f *fakeOutboxRepo) Save(context.Context, *domain.OutboxEvent) error { return nil }
func (f *fakeOutboxRepo) GetUnsentMessages(context.Context, int) ([]*domain.OutboxEvent, error) {
	return f.pending, f.fetchErr
}
func (f *fakeOutboxRepo) MarkAsSent(_ context.Context, id string) error {
	f.sent = append(f.sent, id)
	return nil
}
func (f *fakeOutboxRepo) IncrementRetryCount(_ context.Context, id string) error {
	f.incremented = append(f.incremented, id)
	return nil
}
func (f *fakeOutboxRepo) MarkAsFailed(_ context.Context, id, _ string) error {
	f.failed = append(f.failed, id)
	return nil
}

// fakeProducer คืน error ตาม EventID ที่กำหนดไว้ และจดลำดับที่ถูกส่งจริง
type fakeProducer struct {
	errFor    map[string]error
	published []string
}

func (f *fakeProducer) Send(_ string, _ string, _ []byte, headers map[string]string) error {
	id := headers["EventID"]
	if err := f.errFor[id]; err != nil {
		return err
	}
	f.published = append(f.published, id)
	return nil
}
func (f *fakeProducer) Close() error { return nil }

func event(id, aggregate string, retries int) *domain.OutboxEvent {
	return &domain.OutboxEvent{ID: id, Topic: "order.events", AggregateID: aggregate, EventType: id, RetryCount: retries}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// order #123: CREATED ตามด้วย PAID — Kafka ล่มตอนส่ง CREATED
func TestProcessBatch_BrokerDownStopsBatchWithoutCountingRetry(t *testing.T) {
	repo := &fakeOutboxRepo{pending: []*domain.OutboxEvent{
		event("123-created", "order-123", 0),
		event("123-paid", "order-123", 0),
		event("456-created", "order-456", 0),
	}}
	prod := &fakeProducer{errFor: map[string]error{"123-created": sarama.ErrOutOfBrokers}}
	p := NewOutboxProcessor(repo, prod)

	brokerDown := p.processBatch(context.Background())

	if !brokerDown {
		t.Error("processBatch() = false, want true so Start pauses while Kafka is down")
	}
	if len(prod.published) != 0 {
		t.Errorf("published %v, want nothing after a broker-level error (PAID must not overtake CREATED)", prod.published)
	}
	if len(repo.incremented) != 0 || len(repo.failed) != 0 {
		t.Errorf("incremented=%v failed=%v, want no retry counted for a Kafka outage", repo.incremented, repo.failed)
	}
}

// message ใหญ่เกิน (ส่งกี่รอบก็พัง) — ต้องกัก event ถัดไปของ order เดียวกัน แต่ order อื่นส่งต่อได้
func TestProcessBatch_PermanentErrorHoldsOnlyThatAggregate(t *testing.T) {
	repo := &fakeOutboxRepo{pending: []*domain.OutboxEvent{
		event("123-created", "order-123", 0),
		event("456-created", "order-456", 0),
		event("123-paid", "order-123", 0),
		event("456-paid", "order-456", 0),
	}}
	tooLarge := sarama.ConfigurationError("Attempt to produce message larger than configured Producer.MaxMessageBytes")
	prod := &fakeProducer{errFor: map[string]error{"123-created": tooLarge}}
	p := NewOutboxProcessor(repo, prod)

	if p.processBatch(context.Background()) {
		t.Error("processBatch() = true, want false: a bad message is not a broker outage")
	}
	if want := []string{"456-created", "456-paid"}; !equal(prod.published, want) {
		t.Errorf("published %v, want %v (123-paid must wait for 123-created)", prod.published, want)
	}
	if want := []string{"123-created"}; !equal(repo.incremented, want) {
		t.Errorf("incremented %v, want %v", repo.incremented, want)
	}
}

func TestProcessBatch_PermanentErrorAtMaxRetriesMarksFailed(t *testing.T) {
	repo := &fakeOutboxRepo{pending: []*domain.OutboxEvent{event("123-created", "order-123", 5)}}
	prod := &fakeProducer{errFor: map[string]error{"123-created": sarama.ErrMessageSizeTooLarge}}
	p := NewOutboxProcessor(repo, prod)

	p.processBatch(context.Background())

	if want := []string{"123-created"}; !equal(repo.failed, want) {
		t.Errorf("failed %v, want %v", repo.failed, want)
	}
}

func TestProcessBatch_AllSentInOrder(t *testing.T) {
	repo := &fakeOutboxRepo{pending: []*domain.OutboxEvent{
		event("123-created", "order-123", 0),
		event("123-paid", "order-123", 0),
	}}
	prod := &fakeProducer{}
	p := NewOutboxProcessor(repo, prod)

	if p.processBatch(context.Background()) {
		t.Error("processBatch() = true, want false")
	}
	if want := []string{"123-created", "123-paid"}; !equal(prod.published, want) || !equal(repo.sent, want) {
		t.Errorf("published=%v sent=%v, want both %v", prod.published, repo.sent, want)
	}
}

// DB ล่ม → อ่าน outbox ไม่ได้ ต้องพักแบบ backoff เหมือน Kafka ล่ม
// ไม่งั้นจะ query ซ้ำ + log error ทุก 500ms ตลอดเวลาที่ DB ล่ม
func TestProcessBatch_FetchErrorPauses(t *testing.T) {
	repo := &fakeOutboxRepo{fetchErr: errors.New("dial tcp: connection refused")}
	prod := &fakeProducer{}
	p := NewOutboxProcessor(repo, prod)

	if !p.processBatch(context.Background()) {
		t.Error("processBatch() = false, want true so Start pauses while the database is down")
	}
	if len(prod.published) != 0 {
		t.Errorf("published %v, want nothing when the outbox could not be read", prod.published)
	}
}

func TestAfterBatch_BackoffGrowsCapsAndResets(t *testing.T) {
	p := NewOutboxProcessor(&fakeOutboxRepo{}, &fakeProducer{})
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

	var got []time.Duration
	for i := 0; i < 7; i++ {
		p.afterBatch(now, true)
		got = append(got, p.backoff)
	}
	want := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second, 30 * time.Second}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("backoff sequence %v, want %v", got, want)
		}
	}
	if !p.pausedUntil.Equal(now.Add(30 * time.Second)) {
		t.Errorf("pausedUntil = %v, want now+30s", p.pausedUntil)
	}

	p.afterBatch(now, false)
	if p.backoff != 0 {
		t.Errorf("backoff after success = %s, want reset to 0", p.backoff)
	}
}

func TestIsPermanentSendError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"no brokers", sarama.ErrOutOfBrokers, false},
		{"not leader", sarama.ErrNotLeaderForPartition, false},
		{"network", errors.New("dial tcp: connection refused"), false},
		{"client size check", sarama.ConfigurationError("message too large"), true},
		{"broker size check", sarama.ErrMessageSizeTooLarge, true},
		{"invalid record", sarama.ErrInvalidRecord, true},
	}
	for _, tt := range tests {
		if got := isPermanentSendError(tt.err); got != tt.want {
			t.Errorf("%s: isPermanentSendError(%v) = %v, want %v", tt.name, tt.err, got, tt.want)
		}
	}
}
