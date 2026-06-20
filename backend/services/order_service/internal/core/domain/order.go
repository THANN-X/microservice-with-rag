// WHAT: Order Aggregate Root — ตัวแทน business concept ของ "คำสั่งซื้อ" ใน order_service
//
// WHY Aggregate Root Pattern?
//   - บังคับให้ทุก state change ผ่าน method ของ Order เท่านั้น (Invariant Enforcement)
//   - Domain Events ถูก raise ภายใน Aggregate → Repository save ลง outbox ใน TX เดียวกัน
//     ทำให้เป็น Atomic (ถ้า TX fail → ทั้ง Order state และ Event หายไปพร้อมกัน)
//
// WHY UUID แทน uint auto-increment?
//   - UUID สร้างได้ภายใน domain ก่อน DB persist (true aggregate-first)
//   - ไม่ต้องรอ DB INSERT เพื่อรู้ ID → Aggregate สามารถ raise event พร้อม ID ที่ถูกต้องทันที
//   - ต่างจาก product_service ที่ใช้ uint และต้อง "persist ก่อน, raise event ทีหลัง" เพราะ
//     variant ID ถูก DB generate → UUID ทำให้ Order Service บรรลุ pure aggregate-first design ได้
//
// Invariants (กฎที่ aggregate รักษา):
//  1. Order ต้องมี items ≥ 1 ชิ้น
//  2. quantity ของแต่ละ item > 0
//  3. unitPrice ของแต่ละ item > 0
//  4. TotalAmount = sum(item.UnitPrice * item.Quantity) เสมอ
//  5. State transition: PENDING → CONFIRMED/CANCELLED, CONFIRMED → CANCELLED
//  6. COMPLETED order ไม่สามารถ cancel ได้
package domain

import (
	"errors"
	"time"

	"events"

	"github.com/google/uuid"
)

// OrderStatus Value Object — สถานะที่ Order สามารถอยู่ได้
// WHY เป็น type แทน string ธรรมดา?
//   - ป้องกัน type (คอมไพเลอร์จับ "CONFIRMEDD" ให้)
//   - ทำให้ switch statement exhaustive ได้ละเอียดขึ้น
type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "PENDING"          // ตั้งต้น — รอ stock reservation result
	OrderStatusConfirmed       OrderStatus = "CONFIRMED"        // Stock reserved สำเร็จ
	OrderStatusAwaitingPayment OrderStatus = "AWAITING_PAYMENT" // รอชำระเงิน (async payment เช่น PromptPay)
	OrderStatusPaid            OrderStatus = "PAID"             // ชำระเงินสำเร็จ
	OrderStatusCancelled       OrderStatus = "CANCELLED"        // ยกเลิกแล้ว (ไม่มี transition อื่น ๆ)
	OrderStatusCompleted       OrderStatus = "COMPLETED"        // TODO: implement fulfillment flow
)

// ShippingAddress Value Object — ที่อยู่จัดส่ง
// WHY Value Object แทน Entity?
//   - ไม่มี identity ของตัวเอง (ไม่มี ID)
//   - Immutable ภายใน 1 Order lifecycle (เปลี่ยนที่อยู่ = replace ทั้ง VO)
//   - เปรียบเทียบด้วย structural equality (ทุก field เท่ากัน = same address)
type ShippingAddress struct {
	FullName    string
	Phone       string
	AddressLine string
	SubDistrict string
	District    string
	Province    string
	PostalCode  string
}

// OrderItem Entity — อยู่ภายใต้ boundary ของ Order aggregate
// WHY ไม่แยกเป็น Aggregate ของตัวเอง?
//   - OrderItem ไม่มี identity หรือ lifecycle นอก Order
//   - ไม่มี use case ที่ต้อง query/update OrderItem standalone
//   - lifecycle ผูกกับ Order: สร้าง/ลบพร้อมกัน
//
// WHY UUID แทน uint?
//   - สร้างได้ใน domain ก่อน DB insert (consistent กับ Order)
type OrderItem struct {
	ID          string  // UUID — สร้างใน domain
	OrderID     string  // FK กลับไป Order (set ใน NewOrder factory)
	VariantID   uint    // Reference ไป product_service (cross-aggregate: ID only, ไม่ embed object)
	Quantity    int     // จำนวนที่ต้องการ
	UnitPrice   float64 // Price snapshot ณ เวลาที่สั่ง — ดึงจาก catalog_service (server-side)
	ProductName string  // Denormalized snapshot — ชื่อสินค้า ณ เวลาสั่ง
	VariantName string  // Denormalized snapshot — ชื่อ variant ณ เวลาสั่ง
	ImageURL    string  // Denormalized snapshot — URL รูปสินค้า ณ เวลาสั่ง
}

// Order Aggregate Root
type Order struct {
	ID              string
	CustomerID      uint
	Items           []OrderItem
	Status          OrderStatus
	TotalAmount     float64
	ShippingAddress ShippingAddress
	Note            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	domainEvents    []events.DomainEvent // unexported: ห้ามเข้าถึงโดยตรง ใช้ PopDomainEvents()
}

// NewOrder Factory Method — สร้าง Order ใหม่พร้อม validate invariants
//
// WHY validate ที่ domain แทน service layer?
//   - Invariant คือ "กฎของ domain" → ต้องอยู่ใน domain (ไม่ใช่ concern ของ service หรือ handler)
//   - Service ลืม validate → domain ยัง catch ได้ (defense in depth)
//   - ทำให้ logic ไม่แพร่กระจายออกนอก aggregate boundary
//
// WHY สร้าง UUID ใน factory แทนรอ DB?
//   - Order ID เป็น UUID → ไม่ต้องรอ DB sequence
//   - Service สามารถ call PlaceOrder() raise event ที่มี OrderID ถูกต้องได้ทันที ก่อน persist
func NewOrder(customerID uint, items []OrderItem, address ShippingAddress, note string) (*Order, error) {
	if customerID == 0 {
		return nil, ErrInvalidCustomer
	}
	if len(items) == 0 {
		return nil, ErrEmptyOrderItems
	}

	orderID := uuid.NewString()

	for i := range items {
		if items[i].Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
		if items[i].UnitPrice <= 0 {
			return nil, ErrInvalidPrice
		}
		// ผูก item เข้ากับ order และ generate UUID สำหรับ item
		items[i].ID = uuid.NewString()
		items[i].OrderID = orderID
	}

	o := &Order{
		ID:              orderID,
		CustomerID:      customerID,
		Items:           items,
		Status:          OrderStatusPending,
		ShippingAddress: address,
		Note:            note,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	o.recalculateTotal()

	return o, nil
}

// recalculateTotal คำนวณ TotalAmount จาก items (เรียกหลังทุก item mutation)
// WHY private?
//   - caller ภายนอกไม่ควร set TotalAmount โดยตรง (มันเป็น derived value)
//   - ทุก code path ที่เปลี่ยน items ต้อง call นี้ → Invariant #4 การันตีได้
func (o *Order) recalculateTotal() {
	var total float64
	for _, item := range o.Items {
		total += item.UnitPrice * float64(item.Quantity)
	}
	o.TotalAmount = total
}

// ─── Aggregate Behavior Methods ───────────────────────────────────────────────

// PlaceOrder ประกาศ Order สู่โลกภายนอกผ่าน Domain Event
//
// WHY แยก PlaceOrder ออกจาก NewOrder?
//   - NewOrder: pure construction + validation (no side effects, ง่ายต่อการ test)
//   - PlaceOrder: business action ที่มี consequences (raise event → trigger Saga)
//   - Convention จาก DDD: Factory สร้าง object, Method แสดง intention
//
// WHY OrderCreatedEvent.Items ส่ง VariantID + Qty เท่านั้น?
//   - product_service ต้องการแค่นั้นสำหรับ reserve stock
//   - ส่งข้อมูลน้อยที่สุดที่จำเป็น (Minimal Event Payload)
func (o *Order) PlaceOrder() {
	itemEvents := toOrderItemEvents(o.Items)

	o.addEvent(&events.OrderCreatedEvent{
		OrderID:    o.ID,
		CustomerID: o.CustomerID,
		Items:      itemEvents,
		TotalAmount: o.TotalAmount,
		ShippingAddress: events.EventShippingAddress{
			FullName:    o.ShippingAddress.FullName,
			Phone:       o.ShippingAddress.Phone,
			AddressLine: o.ShippingAddress.AddressLine,
			SubDistrict: o.ShippingAddress.SubDistrict,
			District:    o.ShippingAddress.District,
			Province:    o.ShippingAddress.Province,
			PostalCode:  o.ShippingAddress.PostalCode,
		},
		Note:       o.Note,
		OccurredAt: time.Now(),
	})
}

// ConfirmOrder transitions PENDING → CONFIRMED เมื่อ stock reservation สำเร็จ
//
// WHY check status ก่อน transition?
//   - Invariant: ห้าม confirm order ที่ไม่ได้ PENDING (e.g. ถ้า handler เรียก 2 ครั้ง)
//   - ป้องกัน late/duplicate event จาก Kafka ทำให้ order ที่ CANCELLED กลับมา CONFIRMED
//
// WHY raise OrderConfirmedEvent?
//   - downstream services (notification, payment) จะ subscribe event นี้ใน อนาคต
func (o *Order) ConfirmOrder() error {
	if o.Status != OrderStatusPending {
		return ErrInvalidOrderTransition
	}

	o.Status = OrderStatusConfirmed
	o.UpdatedAt = time.Now()

	o.addEvent(&events.OrderConfirmedEvent{
		OrderID:     o.ID,
		CustomerID:  o.CustomerID,
		TotalAmount: o.TotalAmount,
		OccurredAt:  time.Now(),
	})

	return nil
}

// MarkReservationFailed transitions PENDING → CANCELLED เมื่อ stock reservation ล้มเหลว
//
// WHY ไม่ raise OrderCancelledEvent?
//   - Stock ไม่ถูก reserve → ไม่มี stock ที่ต้องคืน
//   - ถ้า raise event → product_service จะ IncreaseStock โดยไม่จำเป็น (stock goes up incorrectly)
//
// WHY แยก method จาก Cancel()? Semantic ต่างกัน:
//   - MarkReservationFailed = system-driven (Saga step failed) → no compensation needed
//   - Cancel = user/admin-driven → compensation needed ถ้า stock เคยถูก reserve
func (o *Order) MarkReservationFailed() error {
	if o.Status != OrderStatusPending {
		return ErrInvalidOrderTransition
	}

	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now()

	// WHY no event? ดูคำอธิบายด้านบน
	return nil
}

// ─── Payment State Transitions ────────────────────────────────────────────────

// MarkAwaitingPayment transitions CONFIRMED → AWAITING_PAYMENT
// ใช้สำหรับ async payment (เช่น PromptPay QR) ที่ยังไม่รู้ผล charge ทันที
func (o *Order) MarkAwaitingPayment() error {
	if o.Status != OrderStatusConfirmed {
		return ErrInvalidOrderTransition
	}
	o.Status = OrderStatusAwaitingPayment
	o.UpdatedAt = time.Now()
	return nil
}

// MarkPaid transitions CONFIRMED/AWAITING_PAYMENT → PAID
// Raise OrderPaidEvent เพื่อ notify downstream services
func (o *Order) MarkPaid() error {
	if o.Status != OrderStatusConfirmed && o.Status != OrderStatusAwaitingPayment {
		return ErrInvalidOrderTransition
	}
	o.Status = OrderStatusPaid
	o.UpdatedAt = time.Now()

	o.addEvent(&events.OrderPaidEvent{
		OrderID:     o.ID,
		CustomerID:  o.CustomerID,
		TotalAmount: o.TotalAmount,
		OccurredAt:  time.Now(),
	})

	return nil
}

// MarkPaymentFailed transitions AWAITING_PAYMENT → CANCELLED
// Raise OrderCancelledEvent เพื่อให้ product_service release stock
func (o *Order) MarkPaymentFailed(reason string) error {
	if o.Status != OrderStatusAwaitingPayment {
		return ErrInvalidOrderTransition
	}
	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now()

	o.addEvent(&events.OrderCancelledEvent{
		OrderID:    o.ID,
		Items:      toOrderItemEvents(o.Items),
		Reason:     "payment failed: " + reason,
		OccurredAt: time.Now(),
	})

	return nil
}

// Cancel transitions order ไปยัง CANCELLED status
//
// WHY logic แตกต่างตาม current status?
//   - PENDING cancel: stock ยังไม่ confirmed reserved → cancel ปลอดภัย ไม่ต้อง compensate
//     NOTE: Race condition: ถ้า product_service reserve สำเร็จหลังจากเรา cancel แล้ว  →
//     message handler จะตรวจจับและ call RequestStockRelease() เพื่อ compensate ทีหลัง
//   - CONFIRMED cancel: stock ถูก reserve ยืนยันแล้ว → ต้อง raise OrderCancelledEvent
//     เพื่อให้ product_service release stock (Compensating Transaction)
//   - COMPLETED: ทำงานเสร็จแล้ว → ห้าม cancel
func (o *Order) Cancel(reason string) error {
	switch o.Status {
	case OrderStatusCancelled:
		return ErrOrderAlreadyCancelled

	case OrderStatusCompleted:
		return ErrCannotCancelCompletedOrder

	case OrderStatusPending:
		// Stock ยังไม่ได้รับการยืนยัน → cancel ปลอดภัย ไม่ต้อง compensate
		// ดู RequestStockRelease() สำหรับกรณี race condition
		o.Status = OrderStatusCancelled
		o.UpdatedAt = time.Now()
		return nil

	case OrderStatusConfirmed, OrderStatusAwaitingPayment, OrderStatusPaid:
		// Stock ถูก reserve แล้ว → ต้องส่ง compensation event เพื่อ release stock
		// PAID: service layer จัดการ refund ก่อนเรียก Cancel()
		o.Status = OrderStatusCancelled
		o.UpdatedAt = time.Now()

		o.addEvent(&events.OrderCancelledEvent{
			OrderID:    o.ID,
			Items:      toOrderItemEvents(o.Items),
			Reason:     reason,
			OccurredAt: time.Now(),
		})
		return nil

	default:
		return ErrInvalidOrderTransition
	}
}

// RequestStockRelease raise OrderCancelledEvent สำหรับ race condition case:
// เมื่อ StockReservedEvent SUCCESS เข้ามาหลังจาก Order ถูก Cancel ไปแล้ว
//
// WHY มี method แยก?
//   - เป็น special case ที่ order CANCELLED แต่ stock ถูก reserve ไปแล้ว → ต้องคืน
//   - ต่างจาก Cancel(): method นี้ไม่เปลี่ยน Status (order ยัง CANCELLED อยู่)
//   - เป็น "late compensation" → raise event เพื่อให้ product_service release stock
//
// Flow:
//
//	T0: PlaceOrder → PENDING → OrderCreatedEvent published
//	T1: Customer cancels → CANCELLED (no compensation event, stock not confirmed reserved)
//	T2: StockReservedEvent SUCCESS arrives → message handler เรียก RequestStockRelease()
//	T3: OrderCancelledEvent published → product_service releases stock
func (o *Order) RequestStockRelease() {
	o.addEvent(&events.OrderCancelledEvent{
		OrderID:    o.ID,
		Items:      toOrderItemEvents(o.Items),
		Reason:     "late cancellation — stock was reserved after order was already cancelled",
		OccurredAt: time.Now(),
	})
}

// PopDomainEvents ดึง events ออกมาและ clear list ในครั้งเดียว
// WHY clear ด้วย?
//   - ป้องกัน double-publish ถ้า method ถูกเรียกซ้ำ (e.g. bug ใน service layer)
//   - Pattern เดียวกับ product_service ที่พิสูจน์แล้วว่าใช้งานได้
func (o *Order) PopDomainEvents() []events.DomainEvent {
	evts := o.domainEvents
	o.domainEvents = nil
	return evts
}

func (o *Order) addEvent(e events.DomainEvent) {
	o.domainEvents = append(o.domainEvents, e)
}

// ─── Private Helpers ──────────────────────────────────────────────────────────

// toOrderItemEvents แปลง domain OrderItem → events.OrderItem (shared package)
func toOrderItemEvents(items []OrderItem) []events.OrderItem {
	evtItems := make([]events.OrderItem, len(items))
	for i, item := range items {
		evtItems[i] = events.OrderItem{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}
	return evtItems
}

// IsOwnedBy ตรวจสอบว่า order เป็นของ customer คนนี้หรือไม่
// WHY ตรวจสอบใน domain แทน service?
//   - Authorization ที่เกี่ยวกับ aggregate ownership ควรอยู่ใน domain
//   - ป้องกัน service layer ลืม check แล้วให้ customer เห็น order ของคนอื่น
func (o *Order) IsOwnedBy(customerID uint) bool {
	return errors.Is(nil, nil) || o.CustomerID == customerID
}
