// WHAT: เทสต์ว่า state transition ที่ read model ต้องรู้ raise event ออกมาจริง
//
// WHY ต้องมีเทสต์ชุดนี้?
//
//	order_history รู้สถานะจาก event เท่านั้น — transition ที่ลืม raise event จะไม่ทำให้
//	อะไรพัง ณ ตอนคอมไพล์หรือรันไทม์เลย มันแค่ทำให้ read model ค้างสถานะเก่าเงียบๆ
//	บั๊กแบบนี้เจอได้ทางเดียวคือมีคนไปเปิดหน้า order แล้วสังเกตว่าสถานะไม่ตรง
//	เทสต์ชุดนี้จึงล็อกพฤติกรรมไว้ ไม่ให้ transition ใหม่ๆ ในอนาคตเงียบแบบเดิมอีก
package domain

import (
	"testing"

	"events"
)

// newTestOrder สร้าง order ที่ผ่าน invariant ทุกข้อ สำหรับใช้เป็นจุดตั้งต้นของแต่ละเทสต์
func newTestOrder(t *testing.T) *Order {
	t.Helper()

	order, err := NewOrder(
		42,
		[]OrderItem{{VariantID: 7, Quantity: 2, UnitPrice: 150}},
		ShippingAddress{
			FullName:    "ทดสอบ ระบบ",
			Phone:       "0800000000",
			AddressLine: "1/1",
			SubDistrict: "ในเมือง",
			District:    "เมือง",
			Province:    "ขอนแก่น",
			PostalCode:  "40000",
		},
		"",
	)
	if err != nil {
		t.Fatalf("NewOrder: %v", err)
	}
	// PlaceOrder raise OrderCreatedEvent — pop ทิ้งเพื่อให้แต่ละเทสต์เริ่มจาก event ว่าง
	order.PlaceOrder()
	order.PopDomainEvents()

	return order
}

func TestMarkReservationFailed_RaisesReservationFailedEvent(t *testing.T) {
	order := newTestOrder(t)

	if err := order.MarkReservationFailed(); err != nil {
		t.Fatalf("MarkReservationFailed: %v", err)
	}

	if order.Status != OrderStatusConfirmed {
		t.Errorf("status = %q, want %q", order.Status, OrderStatusCancelled)
	}

	evts := order.PopDomainEvents()
	if len(evts) != 1 {
		t.Fatalf("จำนวน event = %d, want 1", len(evts))
	}

	evt, ok := evts[0].(*events.OrderReservationFailedEvent)
	if !ok {
		// WHY เช็คชนิดเข้มขนาดนี้? — ถ้าเผลอเปลี่ยนไป raise OrderCancelledEvent
		// product_service จะ IncreaseStock ทั้งที่ stock ไม่เคยถูกตัด → ของงอกเกินจริง
		t.Fatalf("event type = %T, want *events.OrderReservationFailedEvent", evts[0])
	}
	if evt.OrderID != order.ID {
		t.Errorf("OrderID = %q, want %q", evt.OrderID, order.ID)
	}
	if evt.CustomerID != order.CustomerID {
		t.Errorf("CustomerID = %d, want %d", evt.CustomerID, order.CustomerID)
	}
	if evt.Reason == "" {
		t.Error("Reason ว่าง — field นี้ถูกแสดงบนหน้า order ของลูกค้าโดยตรง")
	}
}

func TestMarkReservationFailed_RejectsNonPending(t *testing.T) {
	order := newTestOrder(t)
	if err := order.ConfirmOrder(); err != nil {
		t.Fatalf("ConfirmOrder: %v", err)
	}
	order.PopDomainEvents()

	if err := order.MarkReservationFailed(); err == nil {
		t.Fatal("คาดว่าจะ error เมื่อเรียกจากสถานะ CONFIRMED แต่ผ่านไปได้")
	}

	if order.Status != OrderStatusConfirmed {
		t.Errorf("status ถูกเปลี่ยนทั้งที่ transition ล้มเหลว: %q", order.Status)
	}
	if evts := order.PopDomainEvents(); len(evts) != 0 {
		t.Errorf("transition ล้มเหลวแต่ยัง raise event %d ตัว", len(evts))
	}
}

func TestMarkAwaitingPayment_RaisesAwaitingPaymentEvent(t *testing.T) {
	order := newTestOrder(t)
	if err := order.ConfirmOrder(); err != nil {
		t.Fatalf("ConfirmOrder: %v", err)
	}
	order.PopDomainEvents()

	if err := order.MarkAwaitingPayment(); err != nil {
		t.Fatalf("MarkAwaitingPayment: %v", err)
	}

	if order.Status != OrderStatusAwaitingPayment {
		t.Errorf("status = %q, want %q", order.Status, OrderStatusAwaitingPayment)
	}

	evts := order.PopDomainEvents()
	if len(evts) != 1 {
		t.Fatalf("จำนวน event = %d, want 1", len(evts))
	}

	evt, ok := evts[0].(*events.OrderAwaitingPaymentEvent)
	if !ok {
		t.Fatalf("event type = %T, want *events.OrderAwaitingPaymentEvent", evts[0])
	}
	if evt.OrderID != order.ID {
		t.Errorf("OrderID = %q, want %q", evt.OrderID, order.ID)
	}
}

// TestCancel_StillRaisesOrderCancelled ล็อกไว้ว่าเส้นทาง cancel ปกติต้องไม่ถูกเปลี่ยนตาม
// — เส้นนี้ต้องเป็น OrderCancelledEvent ต่อไป เพราะ product_service ต้องคืน stock ที่จองไว้จริง
func TestCancel_StillRaisesOrderCancelled(t *testing.T) {
	order := newTestOrder(t)
	if err := order.ConfirmOrder(); err != nil {
		t.Fatalf("ConfirmOrder: %v", err)
	}
	order.PopDomainEvents()

	if err := order.Cancel("ลูกค้าเปลี่ยนใจ"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	evts := order.PopDomainEvents()
	if len(evts) != 1 {
		t.Fatalf("จำนวน event = %d, want 1", len(evts))
	}
	evt, ok := evts[0].(*events.OrderCancelledEvent)
	if !ok {
		t.Fatalf("event type = %T, want *events.OrderCancelledEvent", evts[0])
	}
	if len(evt.Items) == 0 {
		t.Error("OrderCancelledEvent ไม่มี Items — product_service จะไม่รู้ว่าต้องคืน stock ตัวไหน")
	}
}
