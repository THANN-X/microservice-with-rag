package worker

import (
	"context"
	"errors"
	"order_service/internal/core/domain"
	repo "order_service/internal/core/port/repo"
	"testing"
	"time"
)

// fakeOrderRepo จำลอง DB ด้วย map ของ order และจดว่าเขียนอะไรลงไปบ้าง
// embed interface ไว้ให้ method ที่ worker ไม่ได้ใช้ (เรียกแล้ว panic = เผลอไปใช้)
type fakeOrderRepo struct {
	repo.OrderCommandRepository

	stale   []*domain.Order          // ผลของ FindStalePendingOrders
	current map[string]*domain.Order // สถานะล่าสุดใน "DB" ที่ GetOrderByID คืน
	findErr error
	loadErr map[string]error
	// dbStatus คือ status ที่ CAS เห็นตอน UPDATE — ตั้งต่างจาก current เพื่อจำลอง
	// process อื่นเปลี่ยน status หลังเราโหลดไปแล้ว
	dbStatus map[string]domain.OrderStatus

	updated []string
	events  map[string][]string
}

func (f *fakeOrderRepo) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (f *fakeOrderRepo) FindStalePendingOrders(context.Context, time.Duration, int) ([]*domain.Order, error) {
	return f.stale, f.findErr
}

func (f *fakeOrderRepo) GetOrderByID(_ context.Context, id string) (*domain.Order, error) {
	if err := f.loadErr[id]; err != nil {
		return nil, err
	}
	o, ok := f.current[id]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	copied := *o
	return &copied, nil
}

func (f *fakeOrderRepo) UpdateOrderStatusIf(_ context.Context, id string, from, to domain.OrderStatus) (bool, error) {
	status, ok := f.dbStatus[id]
	if !ok {
		status = f.current[id].Status
	}
	if status != from {
		return false, nil
	}
	f.updated = append(f.updated, id)
	return true, nil
}

func (f *fakeOrderRepo) SaveDomainEvents(_ context.Context, o *domain.Order) error {
	if f.events == nil {
		f.events = map[string][]string{}
	}
	for _, evt := range o.PopDomainEvents() {
		f.events[o.ID] = append(f.events[o.ID], evt.EventName())
	}
	return nil
}

func pendingOrder(id string) *domain.Order {
	return &domain.Order{ID: id, CustomerID: 42, Status: domain.OrderStatusPending}
}

func newFakeOrderRepo(orders ...*domain.Order) *fakeOrderRepo {
	f := &fakeOrderRepo{current: map[string]*domain.Order{}, loadErr: map[string]error{}, dbStatus: map[string]domain.OrderStatus{}}
	for _, o := range orders {
		f.current[o.ID] = o
		f.stale = append(f.stale, o)
	}
	return f
}

// ORDER_CREATED ไปจบที่ DLQ → ไม่มีผลจอง stock กลับมา → order ต้องถูกยกเลิกพร้อม event ให้ order_history
func TestCheckStale_ExpiresStuckPendingOrder(t *testing.T) {
	f := newFakeOrderRepo(pendingOrder("order-123"))
	c := NewReservationTimeoutChecker(f, 15*time.Minute)

	c.checkStale(context.Background())

	if want := []string{"order-123"}; !equal(f.updated, want) {
		t.Errorf("updated %v, want %v", f.updated, want)
	}
	// ต้องเป็น RESERVATION_FAILED ไม่ใช่ ORDER_CANCELLED — ถ้าส่ง CANCELLED product จะคืน stock ที่ไม่เคยจอง
	if want := []string{"ORDER_RESERVATION_FAILED"}; !equal(f.events["order-123"], want) {
		t.Errorf("events %v, want %v", f.events["order-123"], want)
	}
}

// ผลจอง stock มาถึงระหว่าง query รายการค้างกับตอนเข้า TX → โหลดใหม่แล้วเห็น CONFIRMED ต้องข้าม
func TestCheckStale_SkipsOrderConfirmedAfterQuery(t *testing.T) {
	f := newFakeOrderRepo(pendingOrder("order-123"))
	f.current["order-123"] = &domain.Order{ID: "order-123", Status: domain.OrderStatusConfirmed}
	c := NewReservationTimeoutChecker(f, 15*time.Minute)

	c.checkStale(context.Background())

	if len(f.updated) != 0 || len(f.events) != 0 {
		t.Errorf("updated=%v events=%v, want nothing for a confirmed order", f.updated, f.events)
	}
}

// HandleStockResult confirm order หลังเราโหลดเสร็จแต่ก่อนเรา UPDATE → CAS ต้องแพ้และไม่ส่ง event
// ถ้าเขียนทับเป็น CANCELLED ได้ stock ที่จองไว้จริงจะค้างตลอดไป
func TestCheckStale_LosesRaceToConfirmWithoutSavingEvents(t *testing.T) {
	f := newFakeOrderRepo(pendingOrder("order-123"))
	f.dbStatus["order-123"] = domain.OrderStatusConfirmed
	c := NewReservationTimeoutChecker(f, 15*time.Minute)

	c.checkStale(context.Background())

	if len(f.updated) != 0 {
		t.Errorf("updated %v, want no write after losing the compare-and-set", f.updated)
	}
	if len(f.events) != 0 {
		t.Errorf("events %v, want none: the order was not cancelled", f.events)
	}
}

// order หนึ่งโหลดไม่ขึ้น (DB สะดุด) ต้องไม่ทำให้ order อื่นในรอบเดียวกันค้างไปด้วย
func TestCheckStale_OneFailureDoesNotBlockOthers(t *testing.T) {
	f := newFakeOrderRepo(pendingOrder("order-123"), pendingOrder("order-456"))
	f.loadErr["order-123"] = errors.New("connection reset")
	c := NewReservationTimeoutChecker(f, 15*time.Minute)

	c.checkStale(context.Background())

	if want := []string{"order-456"}; !equal(f.updated, want) {
		t.Errorf("updated %v, want %v", f.updated, want)
	}
}

func TestCheckStale_FindErrorDoesNothing(t *testing.T) {
	f := newFakeOrderRepo(pendingOrder("order-123"))
	f.findErr = errors.New("db down")
	f.stale = nil
	c := NewReservationTimeoutChecker(f, 15*time.Minute)

	c.checkStale(context.Background())

	if len(f.updated) != 0 {
		t.Errorf("updated %v, want nothing when the query fails", f.updated)
	}
}
