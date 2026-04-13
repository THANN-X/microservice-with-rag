package command

import (
	"cart_service/internal/core/domain"
	repo "cart_service/internal/core/port/repo"
	port "cart_service/internal/core/port/service"
	dto "cart_service/internal/core/port/service/dto"
	"cart_service/internal/core/port/service/mapper"
	"context"
	"errs"
	"errors"
	"logs"
)

type cartCommandService struct {
	cmdRepo repo.CartCommandRepository
}

func NewCartCommandService(cmdRepo repo.CartCommandRepository) port.CartCommandService {
	return &cartCommandService{cmdRepo: cmdRepo}
}

// AddItem — Lazy Creation: ถ้าไม่มีตะกร้าจะสร้างให้อัตโนมัติจาก JWT UserID
// WHY FindOrCreate ก่อน UpsertItem?
//   - ต้องการ cart_id เพื่อ FK ใน cart_items
//   - ON CONFLICT บน (cart_id, variant_id) บวกจำนวนให้อัตโนมัติ
func (s *cartCommandService) AddItem(ctx context.Context, userID uint, req *dto.AddCartItemReq) (*dto.CartRes, error) {
	cart, err := s.cmdRepo.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		logs.Error("AddItem: failed to find or create cart")
		return nil, errs.NewUnexpectedError()
	}

	if err := s.cmdRepo.UpsertItem(ctx, cart.ID, req.VariantID, req.Quantity); err != nil {
		logs.Error("AddItem: failed to upsert item")
		return nil, errs.NewUnexpectedError()
	}

	// Re-fetch เพื่อให้ได้ state ล่าสุดพร้อม items ทั้งหมด
	updatedCart, err := s.cmdRepo.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return nil, errs.NewUnexpectedError()
	}
	return mapper.ToCartRes(updatedCart), nil
}

// RemoveItem — ลบ variant ออกจากตะกร้า
func (s *cartCommandService) RemoveItem(ctx context.Context, userID uint, variantID uint) (*dto.CartRes, error) {
	cart, err := s.cmdRepo.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return nil, errs.NewUnexpectedError()
	}

	if err := s.cmdRepo.RemoveItem(ctx, cart.ID, variantID); err != nil {
		if errors.Is(err, domain.ErrItemNotFound) {
			return nil, errs.NewNotFoundError("cart item not found")
		}
		logs.Error("RemoveItem: failed to remove item")
		return nil, errs.NewUnexpectedError()
	}

	updatedCart, err := s.cmdRepo.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return nil, errs.NewUnexpectedError()
	}
	return mapper.ToCartRes(updatedCart), nil
}

// UpdateItemQuantity — กำหนดจำนวน item ตรงๆ
// WHY quantity=0 → RemoveItem?
//   - UX: ถ้า user ตั้ง qty=0 หมายความว่าต้องการเอาออก ไม่ใช่ error
func (s *cartCommandService) UpdateItemQuantity(ctx context.Context, userID uint, req *dto.UpdateCartItemReq) (*dto.CartRes, error) {
	cart, err := s.cmdRepo.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return nil, errs.NewUnexpectedError()
	}

	if req.Quantity <= 0 {
		if err := s.cmdRepo.RemoveItem(ctx, cart.ID, req.VariantID); err != nil {
			if errors.Is(err, domain.ErrItemNotFound) {
				return nil, errs.NewNotFoundError("cart item not found")
			}
			return nil, errs.NewUnexpectedError()
		}
	} else {
		if err := s.cmdRepo.SetItemQuantity(ctx, cart.ID, req.VariantID, req.Quantity); err != nil {
			if errors.Is(err, domain.ErrItemNotFound) {
				return nil, errs.NewNotFoundError("cart item not found")
			}
			logs.Error("UpdateItemQuantity: failed to set quantity")
			return nil, errs.NewUnexpectedError()
		}
	}

	updatedCart, err := s.cmdRepo.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return nil, errs.NewUnexpectedError()
	}
	return mapper.ToCartRes(updatedCart), nil
}

// ClearCart — ล้างของในตะกร้าทั้งหมด
func (s *cartCommandService) ClearCart(ctx context.Context, userID uint) error {
	cart, err := s.cmdRepo.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return errs.NewUnexpectedError()
	}

	if err := s.cmdRepo.ClearCart(ctx, cart.ID); err != nil {
		logs.Error("ClearCart: failed to clear cart")
		return errs.NewUnexpectedError()
	}
	return nil
}
