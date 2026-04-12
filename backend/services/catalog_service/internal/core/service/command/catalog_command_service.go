package command

import (
	"catalog_service/internal/core/domain"
	repo "catalog_service/internal/core/port/repo"
	serviceport "catalog_service/internal/core/port/service"
	"context"
	"events"
	"logs"
	"time"
)

const consumerID = "catalog-service"

type catalogCommandService struct {
	writeRepo repo.CatalogWriteRepository
	inboxRepo repo.InboxRepository
}

func NewCatalogCommandService(writeRepo repo.CatalogWriteRepository, inboxRepo repo.InboxRepository) serviceport.CatalogCommandService {
	return &catalogCommandService{
		writeRepo: writeRepo,
		inboxRepo: inboxRepo,
	}
}

func (s *catalogCommandService) isProcessed(ctx context.Context, messageID string) (bool, error) {
	return s.inboxRepo.HasProcessed(ctx, messageID, consumerID)
}

func (s *catalogCommandService) markProcessed(ctx context.Context, messageID string) error {
	return s.inboxRepo.MarkProcessed(ctx, &domain.InboxMessage{
		ID:          messageID,
		ConsumerID:  consumerID,
		ProcessedAt: time.Now(),
	})
}

func (s *catalogCommandService) HandleProductCreated(ctx context.Context, messageID string, evt *events.ProductCreatedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	product := &domain.CatalogProduct{
		ProductID:   evt.ProductID,
		Name:        evt.Name,
		Description: evt.Description,
		ImageURLs:   []string{},
		Categories:  []domain.EmbeddedCategory{},
		Variants:    []domain.EmbeddedVariant{},
		IsActive:    true,
		IsDeleted:   false,
		CreatedAt:   evt.OccurredAt,
		UpdatedAt:   evt.OccurredAt,
	}

	if err := s.writeRepo.Upsert(ctx, product); err != nil {
		return err
	}

	return s.markProcessed(ctx, messageID)
}

func (s *catalogCommandService) HandleProductInfoUpdated(ctx context.Context, messageID string, evt *events.ProductInfoUpdatedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	if err := s.writeRepo.UpdateInfo(ctx, evt.ProductID, evt.Name, evt.Description); err != nil {
		return err
	}

	return s.markProcessed(ctx, messageID)
}

func (s *catalogCommandService) HandleProductPriceChanged(ctx context.Context, messageID string, evt *events.ProductPriceChangedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	// TODO: ProductPriceChangedEvent ไม่มี VariantID — ยังไม่สามารถ update ราคาระดับ variant ได้
	// แก้ไขโดยเพิ่ม VariantID เข้า events.ProductPriceChangedEvent ใน pkg/events/events.go
	logs.Warn("catalog: ProductPriceChangedEvent has no VariantID — price update skipped")

	return s.markProcessed(ctx, messageID)
}

func (s *catalogCommandService) HandleProductVariantAdded(ctx context.Context, messageID string, evt *events.ProductVariantAddedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	attrs := make([]domain.VariantAttribute, len(evt.Attributes))
	for i, a := range evt.Attributes {
		attrs[i] = domain.VariantAttribute{Key: a.Key, Value: a.Value}
	}

	variant := domain.EmbeddedVariant{
		VariantID:  evt.VariantID,
		Sku:        evt.Sku,
		Name:       evt.Name,
		Price:      evt.Price,
		Stock:      evt.Stock,
		IsActive:   true,
		Attributes: attrs,
	}

	if err := s.writeRepo.AddVariant(ctx, evt.ProductID, variant); err != nil {
		return err
	}

	return s.markProcessed(ctx, messageID)
}

func (s *catalogCommandService) HandleProductDeleted(ctx context.Context, messageID string, evt *events.ProductDeletedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	if err := s.writeRepo.MarkDeleted(ctx, evt.ProductID); err != nil {
		return err
	}

	return s.markProcessed(ctx, messageID)
}

func (s *catalogCommandService) HandleStockAdjusted(ctx context.Context, messageID string, evt *events.StockAdjustedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	if err := s.writeRepo.UpdateVariantStock(ctx, evt.ProductID, evt.VariantID, evt.NewStock); err != nil {
		return err
	}

	return s.markProcessed(ctx, messageID)
}
