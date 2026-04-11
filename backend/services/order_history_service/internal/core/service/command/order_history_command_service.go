package command

import (
	"context"
	"events"
	"logs"
	"order_history_service/internal/core/domain"
	repo "order_history_service/internal/core/port/repo"
	serviceport "order_history_service/internal/core/port/service"
	"time"
)

const consumerID = "order-history-service"

type orderHistoryCommandService struct {
	writeRepo repo.OrderHistoryWriteRepository
	inboxRepo repo.InboxRepository
}

func NewOrderHistoryCommandService(writeRepo repo.OrderHistoryWriteRepository, inboxRepo repo.InboxRepository) serviceport.OrderHistoryCommandService {
	return &orderHistoryCommandService{
		writeRepo: writeRepo,
		inboxRepo: inboxRepo,
	}
}

func (s *orderHistoryCommandService) isProcessed(ctx context.Context, messageID string) (bool, error) {
	return s.inboxRepo.HasProcessed(ctx, messageID, consumerID)
}

func (s *orderHistoryCommandService) markProcessed(ctx context.Context, messageID string) error {
	return s.inboxRepo.MarkProcessed(ctx, &domain.InboxMessage{
		ID:          messageID,
		ConsumerID:  consumerID,
		ProcessedAt: time.Now(),
	})
}

func (s *orderHistoryCommandService) HandleOrderCreated(ctx context.Context, messageID string, evt *events.OrderCreatedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	items := make([]domain.OrderHistoryItem, len(evt.Items))
	for i, item := range evt.Items {
		items[i] = domain.OrderHistoryItem{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}

	order := &domain.OrderHistory{
		OrderID:     evt.OrderID,
		CustomerID:  evt.CustomerID,
		Status:      "PENDING",
		TotalAmount: evt.TotalAmount,
		Items:       items,
		ShippingAddress: domain.ShippingAddress{
			FullName:    evt.ShippingAddress.FullName,
			Phone:       evt.ShippingAddress.Phone,
			AddressLine: evt.ShippingAddress.AddressLine,
			SubDistrict: evt.ShippingAddress.SubDistrict,
			District:    evt.ShippingAddress.District,
			Province:    evt.ShippingAddress.Province,
			PostalCode:  evt.ShippingAddress.PostalCode,
		},
		Note:      evt.Note,
		CreatedAt: evt.OccurredAt,
		UpdatedAt: evt.OccurredAt,
	}

	if err := s.writeRepo.Upsert(ctx, order); err != nil {
		return err
	}

	logs.Info("order-history: order created — " + evt.OrderID)
	return s.markProcessed(ctx, messageID)
}

func (s *orderHistoryCommandService) HandleOrderConfirmed(ctx context.Context, messageID string, evt *events.OrderConfirmedEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	if err := s.writeRepo.UpdateStatus(ctx, evt.OrderID, "CONFIRMED"); err != nil {
		return err
	}

	logs.Info("order-history: order confirmed — " + evt.OrderID)
	return s.markProcessed(ctx, messageID)
}

func (s *orderHistoryCommandService) HandleOrderCancelled(ctx context.Context, messageID string, evt *events.OrderCancelledEvent) error {
	processed, err := s.isProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	if err := s.writeRepo.MarkCancelled(ctx, evt.OrderID, evt.Reason); err != nil {
		return err
	}

	logs.Info("order-history: order cancelled — " + evt.OrderID)
	return s.markProcessed(ctx, messageID)
}
