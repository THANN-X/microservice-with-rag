// WHAT: OrderHistoryQueryService — read side ของ order_history_service (CQRS)
//
// WHY แยกออกมาจาก order_service?
//   - order_service write ลง PostgreSQL (strong consistency)
//   - order_history_service read จาก MongoDB (denormalized document, fast reads)
//   - ทั้งสองใช้ OrderID เดียวกัน แต่ sync กันผ่าน Kafka event (Eventually Consistent)
//
// Data flow:
//   order_service → [OrderCreated/Confirmed/Cancelled event] → Kafka
//   → orderHistoryCommandService.Handle* → MongoDB upsert
//   → orderHistoryQueryService อ่านจาก MongoDB → return to client
package query

import (
	"context"
	"math"
	"order_history_service/internal/core/domain"
	repo "order_history_service/internal/core/port/repo"
	serviceport "order_history_service/internal/core/port/service"
	"order_history_service/internal/core/port/service/dto"
	"time"

	"errs"
)

type orderHistoryQueryService struct {
	readRepo repo.OrderHistoryReadRepository
}

func NewOrderHistoryQueryService(readRepo repo.OrderHistoryReadRepository) serviceport.OrderHistoryQueryService {
	return &orderHistoryQueryService{readRepo: readRepo}
}

func (s *orderHistoryQueryService) GetOrderByID(ctx context.Context, orderID string, customerID uint) (*dto.OrderHistoryRes, error) {
	order, err := s.readRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// WHY ownership check หลัง load (ไม่ใช้ WHERE id=? AND customer_id=?)?
	//   - ถ้าไม่เจอเพราะ ownership fail → return 401 Unauthorized (ไม่ใช่ 404)
	//   - ป้องกัน information leakage: client ไม่รู้ว่า orderID นั้นมีอยู่หรือเปล่า
	//   - 403/401 vs 404: ถ้า return 404 เมื่อ ownership ไม่ผ่าน = leak ว่า order นั้นมีอยู่จริง
	if order.CustomerID != customerID {
		return nil, errs.NewUnauthorizedError("unauthorized: you do not own this order")
	}

	res := toOrderHistoryRes(*order)
	return &res, nil
}

func (s *orderHistoryQueryService) GetAdminOrderByID(ctx context.Context, orderID string) (*dto.OrderHistoryRes, error) {
	order, err := s.readRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	res := toOrderHistoryRes(*order)
	return &res, nil
}

func (s *orderHistoryQueryService) ListMyOrders(ctx context.Context, customerID uint, req *dto.ListOrderHistoryReq) (*dto.OrderHistoryListRes, error) {
	// WHY ตั้งค่า default ใน Service แทนให้เป็น zero value?
	//   - ควบคุม business default logic ไว้ใน Core Layer → thin handler ไม่ต้องรู้
	//   - limit > 100: ป้องกัน DoS-like (ลูกค้าส่ง limit=999999)
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	filter := domain.OrderHistoryFilter{
		Page:       page,
		Limit:      limit,
		CustomerID: customerID,
		Status:     req.Status,
	}

	orders, total, err := s.readRepo.FindByCustomerID(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]dto.OrderHistoryRes, len(orders))
	for i, o := range orders {
		items[i] = toOrderHistoryRes(o)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &dto.OrderHistoryListRes{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}, nil
}

func (s *orderHistoryQueryService) ListAllOrders(ctx context.Context, req *dto.ListOrderHistoryReq) (*dto.OrderHistoryListRes, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	filter := domain.OrderHistoryAdminFilter{
		Page:   page,
		Limit:  limit,
		Status: req.Status,
	}

	orders, total, err := s.readRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]dto.OrderHistoryRes, len(orders))
	for i, o := range orders {
		items[i] = toOrderHistoryRes(o)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	return &dto.OrderHistoryListRes{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}, nil
}

func (s *orderHistoryQueryService) GetAdminStats(ctx context.Context) (*dto.AdminStatsRes, error) {
	totalOrders, totalRevenue, err := s.readRepo.SumRevenue(ctx)
	if err != nil {
		return nil, err
	}
	return &dto.AdminStatsRes{
		TotalOrders:  totalOrders,
		TotalRevenue: totalRevenue,
	}, nil
}

// toOrderHistoryRes แปลง Domain Object → Response DTO// Subtotal คำนวณ ณ เวลา response (ไม่ได้เก็บใน MongoDB)
//   Subtotal = UnitPrice × Quantity  ← ราคา ณ เวลาที่สั่งซื้อ (snapshot ไม่เปลี่ยนตามราคาปัจจุบัน)
func toOrderHistoryRes(o domain.OrderHistory) dto.OrderHistoryRes {
	items := make([]dto.OrderHistoryItemRes, len(o.Items))
	for i, item := range o.Items {
		items[i] = dto.OrderHistoryItemRes{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Subtotal:  item.UnitPrice * float64(item.Quantity),
		}
	}

	return dto.OrderHistoryRes{
		OrderID:    o.OrderID,
		CustomerID: o.CustomerID,
		Status:     o.Status,
		TotalAmount: o.TotalAmount,
		Items:      items,
		ShippingAddress: dto.ShippingAddressRes{
			FullName:    o.ShippingAddress.FullName,
			Phone:       o.ShippingAddress.Phone,
			AddressLine: o.ShippingAddress.AddressLine,
			SubDistrict: o.ShippingAddress.SubDistrict,
			District:    o.ShippingAddress.District,
			Province:    o.ShippingAddress.Province,
			PostalCode:  o.ShippingAddress.PostalCode,
		},
		Note:         o.Note,
		CancelReason: o.CancelReason,
		CreatedAt:    o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    o.UpdatedAt.Format(time.RFC3339),
	}
}
