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

	if order.CustomerID != customerID {
		return nil, errs.NewUnauthorizedError("unauthorized: you do not own this order")
	}

	res := toOrderHistoryRes(*order)
	return &res, nil
}

func (s *orderHistoryQueryService) ListMyOrders(ctx context.Context, customerID uint, req *dto.ListOrderHistoryReq) (*dto.OrderHistoryListRes, error) {
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
