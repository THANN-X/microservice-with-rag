package command

import (
	"context"
	"encoding/json"
	"errors"
	"errs"
	"events"
	"fmt"
	"logs"
	"product_service/internal/core/domain"
	repo "product_service/internal/core/port/repo"
	service "product_service/internal/core/port/service"
	dto "product_service/internal/core/port/service/dto"
	"product_service/internal/core/port/service/mapper"
	"time"
)

// productCommandService ประกอบด้วย 3 repositories เพราะแต่ละตัวมี responsibility ต่างกัน:
//   - cmdRepo    : เขียน/อ่าน Product aggregate (primary data store)
//   - outboxRepo : บันทึก domain events ลง outbox table (Transactional Outbox Pattern)
//   - inboxRepo  : ตรวจสอบ/บันทึก processed message IDs (Idempotency / Inbox Pattern)
//
// WHY แยกออกเป็น 3 แทนที่จะเป็น repo เดียว → separation of concern + testability
type productCommandService struct {
	cmdRepo    repo.ProductCommandRepository
	outboxRepo repo.OutboxRepository
	inboxRepo  repo.InboxRepository
}

func NewProductCommandService(cmdRepo repo.ProductCommandRepository, outboxRepo repo.OutboxRepository, inboxRepo repo.InboxRepository) service.ProductCommandService {
	return &productCommandService{
		cmdRepo:    cmdRepo,
		outboxRepo: outboxRepo,
		inboxRepo:  inboxRepo}
}

// USE CASE: Create Product (Transactional Outbox Pattern)

// WHY ใช้ RunInTx?
//   - CreateProduct + SaveDomainEvents (outbox) ต้องทำใน transaction เดียวกัน
//   - ถ้า save product สำเร็จแต่ outbox ล้ม → event หาย downstream service ไม่รู้ว่ามีสินค้าใหม่
//   - Transactional Outbox การันตี At-Least-Once delivery
func (s *productCommandService) CreateProduct(ctx context.Context, userID uint, req *dto.CreateProductReq) error {
	// Map Request to Domain Model
	variants := mapper.ToDomain_Variants(req.Variants)
	categories := mapper.ToDomain_Categories(req.CategoryIDs)

	// Create Domain Model
	newProduct, err := domain.NewProduct(
		req.Name,
		req.Description,
		req.ImageURLs,
		variants,
		categories,
	)

	if err != nil {
		return errs.NewValidationError(err.Error()) // ถ้าผิดกฎ พ่นกลับไปหา Client
	}

	// Trigger Business Logic
	newProduct.MarkAsCreated(userID)

	// Transactional Outbox: save the product and its domain event atomically
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Repo assigns the auto-increment ID back into newProduct after insert
		if err := s.cmdRepo.CreateProduct(txCtx, newProduct); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}

// USE CASE: Update Product General Info (Admin)
//
// WHAT: update name, description, categories ของ Product
// WHY Load ก่อน update (Load-Modify-Save pattern)?
//  1. ได้ 404 ทันทีถ้า product ไม่มี (แทน silent 0 rows affected)
//  2. Domain method UpdateInfo() enforce business rule (e.g. name ไม่เป็น empty)
//  3. โครงสร้าง events จาก domain object ได้ถูกต้อง
func (s *productCommandService) UpdateProductGeneralInfo(ctx context.Context, userID uint, req *dto.UpdateProductGeneralInfoReq) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Load Aggregate Root
		product, err := s.cmdRepo.GetProductByID(txCtx, req.ProductID)

		if err != nil {
			logs.Error(err)
			return errs.NewConflictError("failed to get product for update")
		}

		if product == nil {
			return errs.NewNotFoundError(fmt.Sprintf("product with ID %d not found", req.ProductID))
		}

		if err = product.UpdateInfo(req.Name, req.Description); err != nil {
			return errs.NewValidationError(err.Error())
		}

		newCategories := make([]domain.Category, len(req.CategoryIDs))
		for i, id := range req.CategoryIDs {
			newCategories[i] = domain.Category{ID: id}
		}

		product.Categories = newCategories // GORM จะ handle replace ให้ถ้าระบุ setup ถูกต้อง
		product.UpdatedBy = userID

		if err = s.cmdRepo.UpdateProduct(txCtx, product); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}

// USE CASE: Update Variant Price (Admin)
//
// WHAT: เปลี่ยน price ของ variant หนึ่งตัวผ่าน Product aggregate
// WHY เลือกเปลี่ยนผ่าน Aggregate Root แทนที่จะผ่าน Variant โดยตรง?
//   - Domain method UpdateVariantPrice() validate (newPrice >= 0)
//   - Raise ProductPriceChangedEvent เพื่อ downstream รู้ (e.g. Cart update price)
//   - การันตีว่า VariantID อยู่ใน ProductID ที่ระบุจริง (ownership check)
func (s *productCommandService) UpdateVariantPrice(ctx context.Context, userID uint, req *dto.UpdateVariantPriceReq) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Load Aggregate Root
		product, err := s.cmdRepo.GetProductByID(txCtx, req.ProductID)

		if err != nil {
			logs.Error(err)
			return errs.NewConflictError("failed to get product for update")
		}

		if product == nil {
			return errs.NewNotFoundError(fmt.Sprintf("product with ID %d not found", req.ProductID))
		}

		if err := product.UpdateVariantPrice(req.VariantID, req.NewPrice); err != nil {
			return errs.NewValidationError(err.Error())
		}

		product.UpdatedBy = userID

		// Save ผ่าน Root aggregate
		// WHY ส่ง Product ตัวแม่ไป save?
		//   - GORM cascade update Variant ให้ (ถ้า config Association.Replace ไว้)
		//   - ซเตรียม SaveDomainEvents จะ Pop + save outbox เพื่อเบลอไปแจ้ง price change
		// TODO: optimize โดยสร้าง UpdateVariantPrice repo method แยก ถ้าต้องการ SQL targeted update
		if err := s.cmdRepo.UpdateProduct(txCtx, product); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}

// USE CASE: Delete Product (Admin)
//
// WHAT: Soft delete โดย set DeletedAt = NOW()
// WHY Load ก่อนลบ?
//  1. ใช้ aggregate state ในการ raise ProductDeletedEvent → domain event ตำเหน่งได้
//  2. ได้ 404 ถ้า product ไม่มี (GORM Delete ผ่าน RowsAffected=0 ไม่ error)
func (s *productCommandService) DeleteProduct(ctx context.Context, userID uint, productID uint) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Load before delete to capture aggregate state for the domain event
		product, err := s.cmdRepo.GetProductByID(txCtx, productID)
		if err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}
		if product == nil {
			return errs.NewNotFoundError(fmt.Sprintf("product with ID %d not found", productID))
		}

		// Domain Logic (Mark Deleted & raise domain event)
		product.MarkAsDeleted(userID)

		// Soft Delete in DB — repo auto-saves domain events (same as CreateProduct/UpdateProduct)
		if err := s.cmdRepo.DeleteProduct(txCtx, product); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}

// USE CASE: Add Variant (Admin)
//
// WHAT: เพิ่ม Variant ใหม่ (e.g. สีโทนใหม่) เข้าไปใน Product ที่มีอยู่แล้ว
// WHY ต้อง persist variant ก่อนแล้วค่อย raise domain event?
//   - ProductVariantAddedEvent ต้องมี VariantID จริง (DB auto-increment)
//   - AddVariant repo assign ID กลับ → จึงส่งเข้า AddNewVariant() หลัง
func (s *productCommandService) AddVariant(ctx context.Context, userID uint, req *dto.AddVariantReq) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Load Aggregate Root
		product, err := s.cmdRepo.GetProductByID(txCtx, req.ProductID)

		if err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		if product == nil {
			return errs.NewNotFoundError(fmt.Sprintf("product with ID %d not found", req.ProductID))
		}

		attr := make([]domain.VariantAttribute, len(req.AttributeValueIDs))
		for i, a := range req.AttributeValueIDs {
			attr[i] = domain.VariantAttribute{
				ID: a,
			}
		}

		// AttributeValueIDs are linked via the join table in the entity layer
		newVariant := domain.ProductVariant{
			ProductID:   req.ProductID,
			Sku:         req.Sku,
			NameVariant: req.Name,
			Price:       req.Price,
			Stock:       req.Stock,
			IsActive:    true,
			Attributes:  attr,
		}

		// Persist first to obtain the DB-assigned ID before raising the domain event
		if err := s.cmdRepo.AddVariant(txCtx, &newVariant); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		// Call Domain Logic (เพื่อสร้าง Event)
		// ส่ง newVariant ที่มี ID แล้วเข้าไป
		product.AddNewVariant(newVariant)
		product.UpdatedBy = userID

		//(Optional) ถ้า Business ต้องการให้ Product.UpdatedAt เปลี่ยนด้วย
		// ให้สร้าง func เล็กๆ ใน Repo ชื่อ TouchProduct(id) เพื่ออัปเดตแค่ field นี้ field เดียว
		// s.cmdRepo.TouchProduct(txCtx, product.ID)

		// Save Outbox
		// ฟังก์ชัน saveOutboxEvent ใน Repo จะดึง event จาก product.PopDomainEvents() มา save ลง outbox table
		if err := s.cmdRepo.SaveDomainEvents(txCtx, product); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}

// USE CASE: Set Product Active/Inactive (Admin)
// ทำไมถึงไม่ใช้ RunInTx?
//   - เป็น single UPDATE แค่ 1 statement ไม่มี domain event ที่ต้องเขียน outbox
//   - ใช้ RunInTx จะเพิ่ม overhead โดยไม่จำเป็น
func (s *productCommandService) SetProductActive(ctx context.Context, userID uint, productID uint, active bool) error {
	// Verify product exists before issuing the targeted UPDATE
	_, err := s.cmdRepo.GetProductByID(ctx, productID)

	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return errs.NewNotFoundError(fmt.Sprintf("product with ID %d not found", productID))
		}
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	if err := s.cmdRepo.SetProductActive(ctx, productID, active); err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}
	return nil
}

// USE CASE: Set Variant Active/Inactive (Admin)
// ตรวจสอบ variant ownership ผ่าน product aggregate
// เพื่อป้องกัน admin toggle variant ของ product อื่นโดยใช้ known variantID
func (s *productCommandService) SetVariantActive(ctx context.Context, userID uint, productID uint, variantID uint, active bool) error {
	// Load aggregate root เพื่อ verify ทั้ง product และ variant ownership
	product, err := s.cmdRepo.GetProductByID(ctx, productID)

	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return errs.NewNotFoundError(fmt.Sprintf("product with ID %d not found", productID))
		}
		logs.Error(err)
		return errs.NewUnexpectedError()
	}

	// Check variant belongs to this product (domain-level ownership guard)
	found := false

	for _, v := range product.Variants {
		if v.ID == variantID {
			found = true
			break
		}
	}

	if !found {
		return errs.NewNotFoundError(fmt.Sprintf("variant with ID %d not found in product %d", variantID, productID))
	}

	if err := s.cmdRepo.SetVariantActive(ctx, variantID, active); err != nil {
		logs.Error(err)
		return errs.NewUnexpectedError()
	}
	return nil
}

// USE CASE: AdjustStock (Admin)
//
// WHAT: ตั้ง stock เป็นค่า absolute (ไม่ใช่ delta +/-)
// เหมาะกับ: stock take (นับของจริงแล้ว set ตรงๆ), damage write-off
//
// WHY absolute แทน delta?
//   - Stock take คือการ "นับของจริงแล้วตั้งค่าตรงๆ" ไม่ใช่บวกลบ
//   - e.g. นับได้ 48 ชิ้น → set stock = 48 โดยไม่ต้องรู้ว่าก่อนหน้าเป็นเท่าไร
//
// TODO: ถ้าต้องการ delta (+/-qty) ให้แยกเป็น use case ใหม่ เพราะ semantic ต่างกัน
func (s *productCommandService) AdjustStock(ctx context.Context, userID uint, req *dto.AdjustStockReq) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Load Product
		product, err := s.cmdRepo.GetProductByID(txCtx, req.ProductID)
		if err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		if product == nil {
			return errs.NewNotFoundError(fmt.Sprintf("product with ID %d not found", req.ProductID))
		}

		if err := product.AdjustStock(req.VariantID, req.NewStock, req.Reason, userID); err != nil {
			return errs.NewValidationError(err.Error())
		}

		if err := s.cmdRepo.UpdateStock(txCtx, req.VariantID, req.NewStock); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		if err := s.cmdRepo.SaveDomainEvents(txCtx, product); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}

// USE CASE: Reserve Stock (Consumer Side / Saga Step)
//
// WHAT: ตัด stock สำหรับสินค้าใน Order ที่เพิ่งสร้าง
// WHY ใช้ Inbox Pattern?
//   - Kafka at-least-once → OrderCreated อาจถูกส่งซ้ำ
//   - ถ้าไม่บล็อค stock จะติดลบสองเท่า
//
// WHY ใช้ DB-level atomic update (DecreaseStock) แทน domain CheckStockAvailability?
//   - concurrent orders เข้ามาพร้อมกัน → in-memory check จะ race condition
//   - SQL `WHERE stock >= qty` atomic เห็น 0 rows = ไม่พอส่ง InsufficientStockError
func (s *productCommandService) ReserveStock(ctx context.Context, req *dto.ReserveStockReq) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {
		/*protect race condition

		Load Aggregate Root (เพื่อเช็คว่ามีสินค้านี้จริงไหม)
		 product, err := s.cmdRepo.GetProductByID(ctx, req.ProductID)

		 if err != nil {
		 	logs.Error(err)
		 	return errs.NewUnexpectedError()
		 }

		 if product == nil {
			logs.Error(err)
			return errs.NewNotFoundError(fmt.Sprintf("product with ID %d not found", req.ProductID))
		 } */

		// Idempotency Check (Inbox)
		// ตรวจสอบว่า MessageID นี้เคยถูก process สำเร็จไปหรือยัง
		processed, err := s.inboxRepo.HasProcessedMessage(txCtx, req.MessageID)

		if err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		if processed {
			// ถ้าทำไปแล้ว ให้ return nil เพื่อบอก Consumer ว่า "จบงานได้เลย ไม่ต้อง retry"
			return nil
		}

		// Atomically decrement stock; DecreaseStock enforces stock >= qty at the DB level
		for _, item := range req.Items {

			/* Call Domain Method เพื่อเช็คเงื่อนไขต่างๆ
			err = product.CheckStockAvailability(item.VariantID, item.Qty)

			if err != nil {
				logs.Error(err)
				return errs.NewInsufficientStockError(err.Error())
			} */

			// DecreaseStock จะเช็ค stock >= qty ให้เองใน Repo (Atomic Update)
			if err = s.cmdRepo.DecreaseStock(txCtx, item.VariantID, item.Qty); err != nil {

				if errors.Is(err, domain.ErrNoDataModified) {
					logs.Error("Failed to decrease stock: variant not found or stock insufficient")
					return errs.NewInsufficientStockError("stock not enough")
				}

				logs.Error(err)
				return errs.NewUnexpectedError()
			}
		}

		eventItems := make([]events.ReservedItem, len(req.Items))
		for i, item := range req.Items {
			eventItems[i] = events.ReservedItem{
				VariantID: item.VariantID,
				Qty:       item.Qty,
			}
		}

		/*Save Outbox (Manual Construction)
		เราทำที่ Service เพราะต้องการรวมผลลัพธ์ว่า "Order นี้จองสำเร็จแล้ว" เป็น 1 Event */
		evt := &events.StockReservedEvent{
			OrderID:    req.OrderID,   // ส่งกลับไปบอกว่า Order ไหนสำเร็จ
			MessageID:  req.MessageID, // Ref กลับไปหา Saga
			Status:     "SUCCESS",     // บอกผลลัพธ์
			Items:      eventItems,    // แนบรายการไปด้วย (เผื่อใช้)
			OccurredAt: time.Now(),
		}

		payloadBytes, err := json.Marshal(evt)

		if err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		/* สร้าง Event Object
		AggregateID: ใช้ req.MessageID เพื่อผูกกลับไปหา Order Saga เดิม หรือใช้ ProductID ถ้าอยาก tracking รายตัว (แต่เคสนี้มาเป็น list ใช้ OrderID/MessageID เหมาะกว่า) */

		outboxMessage := domain.NewOutboxMessage(
			"stock.events",
			fmt.Sprintf("order-%s", req.OrderID), // AggregateID (Ref กลับไปหา Saga/Order)
			"STOCK",                              // AggregateType
			evt.EventName(),                      // EventType (Success)
			string(payloadBytes),
		)

		if err = s.outboxRepo.Save(txCtx, outboxMessage); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		inboxEvent := &domain.InboxEvent{
			ID:         req.MessageID,
			ConsumerID: "product_service_stock", // ระบุชื่อ Consumer Group
		}

		if err = s.inboxRepo.SaveProcessedMessage(txCtx, inboxEvent); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}

// USE CASE: Release Stock (Compensating Transaction / Saga Rollback)
//
// WHAT: คืน stock กลับเมื่อ Order ถูก Cancel หรือ Payment ล้มเหลว
// WHY ต้องเป็น compensating transaction?
//   - ระบบนี้ใช้ Choreography Saga ในการจัดการ distributed transaction
//   - ถ้า Payment ล้ม Order Service จะส่ง OrderCancelled event มา
//   - Product Service รับ event แล้วต้องคืน stock ที่จองไว้กลับ
//   - Inbox Pattern ป้องกัน ReleaseStock ทำซ้ำ (เหตุ Kafka อาจส่ง event ซ้ำ)
func (s *productCommandService) ReleaseStock(ctx context.Context, req *dto.ReserveStockReq) error {
	return s.cmdRepo.RunInTx(ctx, func(txCtx context.Context) error {

		/* Load Aggregate Root (เพื่อเช็คว่ามีสินค้านี้จริงไหม)
		 product, err := s.cmdRepo.GetProductByID(ctx, req.ProductID)

		 if err != nil {
		 	logs.Error(err)
		 	return errs.NewUnexpectedError()
		 }

		 if product == nil {
			logs.Error(err)
			return errs.NewNotFoundError(fmt.Sprintf("product with ID %d not found", req.ProductID))
		}

		 Idempotency Check (Inbox Pattern)
		 เราต้องเช็ค MessageID ของ Event Cancel/Release นี้ (ไม่ใช่ MessageID ตอนจอง คนละ ID กัน) */

		processed, err := s.inboxRepo.HasProcessedMessage(txCtx, req.MessageID)

		if err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		if processed {
			return nil
		}

		for _, item := range req.Items {
			if err = s.cmdRepo.IncreaseStock(txCtx, item.VariantID, item.Qty); err != nil {

				if errors.Is(err, domain.ErrNoDataModified) {
					logs.Error("Failed to increase stock: variant not found")
					return errs.NewInsufficientStockError("failed to release stock")
				}

				logs.Error(err)
				return errs.NewUnexpectedError()
			}
		}

		eventItems := make([]events.ReservedItem, len(req.Items))

		for i, item := range req.Items {
			eventItems[i] = events.ReservedItem{
				VariantID: item.VariantID,
				Qty:       item.Qty,
			}
		}

		evt := &events.StockReleasedEvent{
			OrderID:    req.OrderID,
			MessageID:  req.MessageID,
			Status:     "SUCCESS",
			Items:      eventItems,
			OccurredAt: time.Now(),
		}

		payloadBytes, err := json.Marshal(evt)

		if err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		outboxMessage := domain.NewOutboxMessage(
			"stock.events",  // Topic
			req.MessageID,   // AggregateID
			"STOCK",         // AggregateType
			evt.EventName(), // EventType (Compensate Success)
			string(payloadBytes),
		)

		if err = s.outboxRepo.Save(txCtx, outboxMessage); err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		inboxEvent := &domain.InboxEvent{
			ID:         req.MessageID,
			ConsumerID: "product_service_stock",
		}

		err = s.inboxRepo.SaveProcessedMessage(txCtx, inboxEvent)

		if err != nil {
			logs.Error(err)
			return errs.NewUnexpectedError()
		}

		return nil
	})
}
