package port

import (
	"context"
	dto "product_service/internal/core/port/service/dto"
)

type AttributeCommandService interface {
	CreateAttribute(ctx context.Context, req *dto.CreateAttributeReq) error
	UpdateAttribute(ctx context.Context, req *dto.UpdateAttributeReq) error
	DeleteAttribute(ctx context.Context, id uint) error

	CreateAttributeValue(ctx context.Context, req *dto.CreateAttributeValueReq) error
	DeleteAttributeValue(ctx context.Context, id uint) error
}

type AttributeQueryService interface {
	GetAllAttributes(ctx context.Context) ([]dto.AttributeRes, error)
	GetAttributeByID(ctx context.Context, id uint) (*dto.AttributeRes, error)
}
