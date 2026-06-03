package mapper

import (
	"cart_service/internal/core/domain"
	"cart_service/internal/core/port/service/dto"
)

func ToCartRes(cart *domain.Cart) *dto.CartRes {
	items := make([]dto.CartItemRes, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = dto.CartItemRes{
			VariantID:   item.VariantID,
			Quantity:    item.Quantity,
			ProductName: item.ProductName,
			VariantName: item.VariantName,
			Price:       item.Price,
			ImageURL:    item.ImageURL,
			AddedAt:     item.AddedAt,
			UpdatedAt:   item.UpdatedAt,
		}
	}
	return &dto.CartRes{
		CartID:    cart.ID,
		UserID:    cart.UserID,
		Items:     items,
		CreatedAt: cart.CreatedAt,
		UpdatedAt: cart.UpdatedAt,
	}
}
