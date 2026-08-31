// WHAT: One-off backfill — ยัด event ที่ "ไม่เคยถูกประกาศ" ย้อนหลังลง outbox
//
// WHY ต้องมี?
//
//	order_history เป็น read model ที่รู้สถานะจาก event เท่านั้น ไม่เคยถาม order_service ย้อนหลัง
//	ก่อนหน้านี้มี state transition 2 ตัวที่เปลี่ยนสถานะแล้วไม่ raise event เลย:
//	  - MarkReservationFailed (ของไม่พอ → CANCELLED)
//	  - MarkAwaitingPayment   (ออก QR → AWAITING_PAYMENT)
//	order ที่ผ่านสองทางนี้ไปแล้วจะค้างสถานะเก่าใน read model ถาวร — ไม่มี reconcile job
//	มาซ่อมให้ และ replay Kafka ก็ไม่ช่วย เพราะ event ไม่เคยถูกส่งตั้งแต่แรก
//
// WHY ยัดลง outbox แทนแก้ MongoDB ตรงๆ?
//   - ใช้ pipeline เดิมทั้งเส้น (OutboxProcessor → Kafka → order_history) ไม่ต้องต่อ Mongo
//     ไม่ต้องรู้ credential ฝั่งนั้น และได้ audit trail ใน outbox ว่าซ่อมอะไรไปบ้าง
//   - ถ้า order_history ล่มอยู่ event จะรอบน Kafka แล้วค่อย apply เมื่อมันกลับมา
//
// SAFETY: event ที่ยัดเป็นชนิดที่ product_service ไม่ consume (ORDER_RESERVATION_FAILED /
//
//	ORDER_AWAITING_PAYMENT / ORDER_PAID) → ไม่มีทางไป trigger stock release ซ้ำ
//	ห้ามเปลี่ยนไปยัด ORDER_CANCELLED เด็ดขาด — product_service จะ IncreaseStock ทันที
//
// วิธีใช้:
//
//	go run ./cmd/backfill              # dry-run — แค่รายงาน ไม่เขียนอะไร
//	go run ./cmd/backfill -apply       # เขียนจริง
//	go run ./cmd/backfill -resync-paid # รวมการ re-emit ORDER_PAID (ดู WHY ที่ scanPaid)
package main

import (
	"encoding/json"
	"events"
	"flag"
	"log"
	"time"

	"order_service/internal/adapter/repository/postgres/entity"
	"order_service/internal/config"
	"order_service/internal/core/domain"

	"gorm.io/gorm"
)

// orderRow คือ projection เล็กๆ พอสำหรับสร้าง event — ไม่ต้องโหลด aggregate เต็ม
type orderRow struct {
	ID          string
	CustomerID  uint
	TotalAmount float64 // ใช้เฉพาะ scanPaid — query อื่นไม่ได้ SELECT มา จึงเป็น 0
	UpdatedAt   time.Time
}

// pendingEvent จับคู่ event ที่จะยัด กับ order ต้นทาง เพื่อใช้ทั้งตอนรายงานและตอนเขียน
type pendingEvent struct {
	OrderID   string
	EventType string
	Payload   string
}

func main() {
	apply := flag.Bool("apply", false, "เขียนลง outbox จริง (ค่าเริ่มต้นคือ dry-run เฉยๆ)")
	resyncPaid := flag.Bool("resync-paid", false, "re-emit ORDER_PAID ของทุก order ที่จ่ายแล้ว")
	flag.Parse()

	cfg := config.Loadconfig()
	db := config.OpenDatabase(cfg.GetDSN())

	var all []pendingEvent
	all = append(all, scanReservationFailed(db)...)
	all = append(all, scanAwaitingPayment(db)...)
	if *resyncPaid {
		all = append(all, scanPaid(db)...)
	}

	if len(all) == 0 {
		log.Println("ไม่พบ order ที่ต้องซ่อม — read model ตรงกับ order_service อยู่แล้ว")
		return
	}

	log.Printf("พบ %d event ที่ต้องยัดย้อนหลัง:", len(all))
	counts := map[string]int{}
	for _, e := range all {
		counts[e.EventType]++
		log.Printf("  %-26s order=%s", e.EventType, e.OrderID)
	}
	for typ, n := range counts {
		log.Printf("สรุป: %s = %d ใบ", typ, n)
	}

	if !*apply {
		log.Println()
		log.Println("นี่คือ dry-run — ยังไม่มีอะไรถูกเขียน รันซ้ำด้วย -apply เพื่อลงจริง")
		return
	}

	// WHY ยัดทั้งชุดใน TX เดียว?
	//   - ถ้าเขียนไปได้ครึ่งเดียวแล้วล้ม จะแยกไม่ออกว่าใบไหนยัดไปแล้ว ต้องมานั่งไล่เอง
	//   - จำนวน row ระดับหลักสิบ/ร้อย TX เดียวสบายๆ
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, e := range all {
			outboxMsg := domain.NewOutboxMessage("order.events", e.OrderID, "ORDER", e.EventType, e.Payload)
			if err := tx.Create(entity.ToOutboxEventEntity(outboxMsg)).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Fatalf("ยัด event ไม่สำเร็จ ไม่มีอะไรถูกเขียน: %v", err)
	}

	log.Printf("ยัด %d event ลง outbox แล้ว — OutboxProcessor จะส่งออกภายในไม่กี่วินาที", len(all))
}

// scanReservationFailed หา order ที่ถูก cancel แต่ไม่เคยมี cancel event ออกไปเลย
//
// WHY ถึงมั่นใจว่าใบพวกนี้คือ "ของไม่พอ"?
//
//	ทางที่ทำให้ order เป็น CANCELLED มีอยู่ 4 ทาง และ 3 ทาง (ลูกค้า cancel / admin cancel /
//	payment timeout) raise OrderCancelledEvent เสมอ เหลือทางเดียวที่เงียบคือ
//	MarkReservationFailed → order ที่ CANCELLED แต่ outbox ไม่มี cancel event เลย
//	จึงมาจากทางนั้นทางเดียว
//
// WHY เช็ค ORDER_RESERVATION_FAILED ด้วย?
//
//	เพื่อให้รันสคริปต์ซ้ำได้ไม่ยัดซ้ำ (รอบสองจะเห็น event ที่รอบแรกยัดไว้แล้ว)
func scanReservationFailed(db *gorm.DB) []pendingEvent {
	var rows []orderRow
	err := db.Raw(`
		SELECT o.id, o.customer_id, o.updated_at
		FROM orders o
		WHERE o.status = 'CANCELLED'
		  AND NOT EXISTS (
		      SELECT 1 FROM outbox_event e
		      WHERE e.aggregate_id = o.id
		        AND e.event_type IN ('ORDER_CANCELLED', 'ORDER_RESERVATION_FAILED')
		  )
		ORDER BY o.created_at
	`).Scan(&rows).Error
	if err != nil {
		log.Fatalf("query CANCELLED ไม่สำเร็จ: %v", err)
	}

	out := make([]pendingEvent, 0, len(rows))
	for _, r := range rows {
		// WHY OccurredAt = updated_at ไม่ใช่ time.Now()?
		//   - updated_at คือเวลาที่ transition เกิดขึ้นจริง การใส่เวลาปัจจุบันจะทำให้
		//     event โกหกว่าเพิ่งเกิด ทั้งที่ order ตายไปนานแล้ว
		out = append(out, marshalEvent(r.ID, &events.OrderReservationFailedEvent{
			OrderID:    r.ID,
			CustomerID: r.CustomerID,
			Reason:     "สินค้าไม่เพียงพอ",
			OccurredAt: r.UpdatedAt,
		}))
	}
	return out
}

// scanAwaitingPayment หา order ที่รอชำระเงินอยู่แต่ไม่เคยประกาศสถานะนี้ออกไป
func scanAwaitingPayment(db *gorm.DB) []pendingEvent {
	var rows []orderRow
	err := db.Raw(`
		SELECT o.id, o.customer_id, o.updated_at
		FROM orders o
		WHERE o.status = 'AWAITING_PAYMENT'
		  AND NOT EXISTS (
		      SELECT 1 FROM outbox_event e
		      WHERE e.aggregate_id = o.id
		        AND e.event_type = 'ORDER_AWAITING_PAYMENT'
		  )
		ORDER BY o.created_at
	`).Scan(&rows).Error
	if err != nil {
		log.Fatalf("query AWAITING_PAYMENT ไม่สำเร็จ: %v", err)
	}

	out := make([]pendingEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, marshalEvent(r.ID, &events.OrderAwaitingPaymentEvent{
			OrderID:    r.ID,
			CustomerID: r.CustomerID,
			OccurredAt: r.UpdatedAt,
		}))
	}
	return out
}

// scanPaid re-emit ORDER_PAID ของทุก order ที่จ่ายแล้ว (ต้องสั่ง -resync-paid เอง)
//
// WHY ต้องแยกเป็น flag ไม่ทำอัตโนมัติ?
//
//	เคสนี้ต่างจากสองตัวบน: ORDER_PAID ถูก "ประกาศออกไปแล้ว" (มี row ใน outbox) แต่ฝั่ง
//	order_history เพิ่งมามี handler รับทีหลัง ของที่ส่งไปก่อนหน้านั้นเลยตกไปเฉยๆ
//	→ ดูจาก outbox อย่างเดียวแยกไม่ออกว่าใบไหน apply แล้วหรือยัง ต้อง re-emit ทั้งหมด
//	   ซึ่งจะแตะ order ที่สถานะถูกอยู่แล้วด้วย (ไม่เสียหาย แต่ไม่ควรทำโดยไม่ตั้งใจ)
//
// SAFETY: ORDER_PAID มี consumer เดียวคือ order_history และ handler เป็น UpdateStatus
//
//	(idempotent) — product_service ไม่ consume event ชนิดนี้เลย
func scanPaid(db *gorm.DB) []pendingEvent {
	var rows []orderRow
	err := db.Raw(`
		SELECT o.id, o.customer_id, o.total_amount, o.updated_at
		FROM orders o
		WHERE o.status = 'PAID'
		ORDER BY o.created_at
	`).Scan(&rows).Error
	if err != nil {
		log.Fatalf("query PAID ไม่สำเร็จ: %v", err)
	}

	out := make([]pendingEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, marshalEvent(r.ID, &events.OrderPaidEvent{
			OrderID:     r.ID,
			CustomerID:  r.CustomerID,
			TotalAmount: r.TotalAmount,
			OccurredAt:  r.UpdatedAt,
		}))
	}
	return out
}

func marshalEvent(orderID string, evt events.DomainEvent) pendingEvent {
	payload, err := json.Marshal(evt)
	if err != nil {
		log.Fatalf("marshal event ของ order %s ไม่สำเร็จ: %v", orderID, err)
	}
	return pendingEvent{
		OrderID:   orderID,
		EventType: evt.EventName(),
		Payload:   string(payload),
	}
}
