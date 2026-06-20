// WHAT: OrderQueryService implementation — read-only use cases (ไม่มี side effects)
// WHY แยก Query Service จาก Command Service?
//   - CQRS: Command และ Query มี concern ต่างกัน → แยก ทำให้ง่ายต่อการ scale แต่ละ side
//   - Query service ไม่ต้องการ TX (read-only) → ง่ายกว่า
//   - ในอนาคต Query service อาจ read จาก read replica หรือ cache layer
package query

import (
	"context"
	"errors"
	"errs"
	"fmt"
	"logs"
	"order_service/internal/core/domain"
	repo "order_service/internal/core/port/repo"
	service "order_service/internal/core/port/service"
	dto "order_service/internal/core/port/service/dto"
	"order_service/internal/core/port/service/mapper"
)

// NOTE: ListMyOrders / ListAllOrders ถูกย้ายไป order_history_service (CQRS read side)

type orderQueryService struct {
	queryRepo repo.OrderQueryRepository
}

func NewOrderQueryService(queryRepo repo.OrderQueryRepository) service.OrderQueryService {
	return &orderQueryService{queryRepo: queryRepo}
}

// GetOrderByID ดึง Order เดี่ยวพร้อมตรวจ ownership
// WHY ownership check ใน service แทน handler?
//   - Service อยู่ใกล้ data → เห็น order.CustomerID → check ทันทีหลัง load
//   - Handler ไม่ควร logic ที่ depend on data content
// WHY ไม่ใช้ WHERE id=? AND customer_id=? ใน repo?
//   - ถ้า not found เพราะ ownership fail → client ได้ 403 (ไม่ใช่ 404)
//     (prevent information leakage: attacker รู้ว่า order ID มีอยู่จริง)
//   - แต่ถ้า not found เพราะ ID ไม่มีจริง → 404
//   - ต้อง Load ก่อนแล้วค่อย check เพื่อ return HTTP code ที่ถูกต้อง
func (s *orderQueryService) GetOrderByID(ctx context.Context, orderID string, customerID uint) (*dto.OrderRes, error) {
	order, err := s.queryRepo.FindByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			return nil, errs.NewNotFoundError(fmt.Sprintf("order %s not found", orderID))
		}
		logs.Error(err)
		return nil, errs.NewUnexpectedError()
	}

	// Ownership check: customer ต้องเป็นเจ้าของ order
	if order.CustomerID != customerID {
		// WHY return 403 (Forbidden) แทน 404?
		//   - Return 404 จะ "leak" ข้อมูลว่า order ID นี้มีอยู่จริง (IDOR vulnerability)
		//   - 403 บอกว่า "order มีอยู่แต่คุณไม่มีสิทธิ์" → ชัดเจนกว่าและปลอดภัยกว่า 404 ในกรณีนี้
		//   - NOTE: ถ้า pattern ต้องการ deny knowledge → ใช้ 404 แทน (security tradeoff)
		return nil, errs.NewForbiddenError("you do not have permission to view this order")
	}

	return mapper.ToOrderRes(order), nil
}

// NOTE: ListMyOrders / ListAllOrders ถูกย้ายไป order_history_service (CQRS read side)
// order_service เก็บเฉพาะ GetOrderByID สำหรับ sync response หลัง PlaceOrder
