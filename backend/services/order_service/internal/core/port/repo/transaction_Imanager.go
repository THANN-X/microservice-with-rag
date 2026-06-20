package port

import "context"

// TransactionManager กำหนด contract สำหรับการทำงานใน DB transaction
// WHY interface แทน concrete type?
//   - ทำให้ core (service/domain) ไม่ผูกกับ GORM หรือ SQL driver โดยตรง
//   - ง่ายต่อการ mock ใน unit test
type TransactionManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
