package port

import "context"

type ProductCommandService interface {
	CreateProduct(ctx context.Context)
}
