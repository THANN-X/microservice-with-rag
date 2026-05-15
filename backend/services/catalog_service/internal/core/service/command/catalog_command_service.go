// WHAT: CatalogCommandService — event consumer ที่ sync product data ลง MongoDB (write side)
//
// WHY ต้องมี catalog_service แยกจาก product_service?
//   - product_service เป็น source of truth (PostgreSQL, normalized)
//   - catalog_service เป็น read-optimized view สำหรับ shop/search (MongoDB, denormalized)
//     → variant + category embed อยู่ใน document เดียว → ค้นหาเร็วโดยไม่ต้อง JOIN
//   - CQRS: write ผ่าน product_service → catalog_service sync ผ่าน Kafka event
//
// Inbox Pattern (เหมือน order_history_service):
//   - ทุก handler ตรวจ messageID ก่อน + mark processed หลัง
//   - consumerID = "catalog-service" (namespace ไม่ซ้ำกับ consumer อื่น)
//
// Events ที่ handle:
//   ProductCreatedEvent      → Upsert empty product doc
//   ProductInfoUpdatedEvent  → UpdateInfo (name, description)
//   ProductPriceChangedEvent → skipped (TODO: fix event ให้มี VariantID ก่อน)
//   ProductVariantAddedEvent → AddVariant (embed variant ใน doc)
//   ProductDeletedEvent      → MarkDeleted
//   StockAdjustedEvent       → UpdateVariantStock
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

// isProcessed ตรวจสอบ Inbox/Idempotency: ถ้า messageID นี้ถูก process ไปแล้วหรือยัง
// WHY ต้องตรวจก่อนทุก event handler?
//   - Kafka guarantee At-Least-Once delivery → message อาจถูกส่งซ้ำ (network retry, consumer rebalance)
//   - ถ้าไม่ตรวจ → Upsert/Update ซ้ำ → MongoDB catalog doc อาจ corrupt
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

// HandleProductCreated สร้าง catalog document ใหม่ใน MongoDB (เริ่มต้นด้วยข้อมูลพื้นฐาน)
// WHY Upsert แทน Insert?
//   - Idempotent: ถ้า retry แล้ว doc เดิมมีอยู่ → overwrite แทนที่จะ error
// HOW: Variants และ Categories เริ่มต้นเป็น empty slice → embed เพิ่มเติมเมื่อมี event ถัดมา
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

// HandleProductVariantAdded embed variant เข้าไปใน catalog document
// HOW: event.Attributes ([]Key/Value) → domain.VariantAttribute → EmbeddedVariant → writeRepo.AddVariant
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

func (s *catalogCommandService) HandleProductImagesUpdated(ctx context.Context, messageID string, evt *events.ProductImagesUpdatedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	if err := s.writeRepo.UpdateProductImages(ctx, evt.ProductID, evt.ImageURLs); err != nil {
		return err
	}

	return s.markProcessed(ctx, messageID)
}

func (s *catalogCommandService) HandleProductVariantImagesUpdated(ctx context.Context, messageID string, evt *events.ProductVariantImagesUpdatedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	if err := s.writeRepo.UpdateVariantImages(ctx, evt.ProductID, evt.VariantID, evt.ImageURLs); err != nil {
		return err
	}

	return s.markProcessed(ctx, messageID)
}
