package query

import (
	"cart_service/internal/core/domain"
	repo "cart_service/internal/core/port/repo"
	port "cart_service/internal/core/port/service"
	dto "cart_service/internal/core/port/service/dto"
	"cart_service/internal/core/port/service/mapper"
	"context"
	"errs"
	"errors"
)

type cartQueryService struct {
	queryRepo repo.CartQueryRepository
}

func NewCartQueryService(queryRepo repo.CartQueryRepository) port.CartQueryService {
	return &cartQueryService{queryRepo: queryRepo}
}

// GetCart คืนตะกร้าของ user
// WHY คืน empty cart แทน 404 เมื่อไม่พบ?
//   - Lazy Creation philosophy: cart ถูกสร้างเมื่อมีการ mutation ครั้งแรก
//   - GET ไม่ควร trigger side effect (creation)
//   - Frontend รับ empty items ได้โดยไม่ต้อง handle 404 แยก
func (s *cartQueryService) GetCart(ctx context.Context, userID uint) (*dto.CartRes, error) {
	cart, err := s.queryRepo.GetCartByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return &dto.CartRes{
				UserID: userID,
				Items:  []dto.CartItemRes{},
			}, nil
		}
		return nil, errs.NewUnexpectedError()
	}
	return mapper.ToCartRes(cart), nil
}
