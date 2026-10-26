// WHAT: package messaging รวม Kafka consumer ที่ retry + ส่ง Dead Letter Queue (DLQ) ให้ทุก service ใช้ร่วมกัน
// WHY: ก่อนหน้านี้ทุก consumer log error แล้ว MarkMessage ต่อทันที
//   - DB ล่มแค่ 2 วินาที → event ที่เข้ามาช่วงนั้นหายถาวร → ข้อมูลข้าม service ไม่ตรงกัน (เช่น stock ไม่ถูก reserve)
//   - message พังก็หายเงียบ ไม่มีที่ให้ย้อนกลับไปดูว่าพังเพราะอะไร
//   - โค้ด ConsumeClaim ซ้ำกัน 4 service → แก้ที่เดียวแล้วได้ทุกตัว
package messaging

import "errors"

// permanentError ห่อ error ที่ retry กี่รอบก็พังแบบเดิม (เช่น JSON ผิด, ไม่มี event type)
type permanentError struct {
	err error
}

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

// Permanent บอก consumer ว่า error นี้ไม่ต้อง retry ให้ส่งเข้า DLQ ทันที
// WHY: retry message ที่ parse ไม่ได้ = เสียเวลา + block partition เปล่าๆ
//
//	error อื่นที่ไม่ได้ห่อ (DB timeout, network) ถือว่าชั่วคราว → retry ก่อน
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// IsPermanent เช็คว่า err (หรือ error ที่ถูก wrap อยู่ข้างใน) ถูกห่อด้วย Permanent หรือไม่
func IsPermanent(err error) bool {
	var p *permanentError
	return errors.As(err, &p)
}
