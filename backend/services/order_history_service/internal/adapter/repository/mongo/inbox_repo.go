package mongo

import (
	"context"
	"order_history_service/internal/core/domain"
	repo "order_history_service/internal/core/port/repo"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const inboxCollection = "order_history_inbox"

type inboxDocument struct {
	ID          string    `bson:"_id"`
	ConsumerID  string    `bson:"consumer_id"`
	ProcessedAt time.Time `bson:"processed_at"`
}

type inboxRepository struct {
	col *mongo.Collection
}

func NewInboxRepository(db *mongo.Database) repo.InboxRepository {
	return &inboxRepository{col: db.Collection(inboxCollection)}
}

func (r *inboxRepository) HasProcessed(ctx context.Context, messageID, consumerID string) (bool, error) {
	filter := bson.M{"_id": compositeID(messageID, consumerID)}
	count, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *inboxRepository) MarkProcessed(ctx context.Context, msg *domain.InboxMessage) error {
	doc := inboxDocument{
		ID:          compositeID(msg.ID, msg.ConsumerID),
		ConsumerID:  msg.ConsumerID,
		ProcessedAt: msg.ProcessedAt,
	}

	_, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return err
	}
	return nil
}

func compositeID(messageID, consumerID string) string {
	return consumerID + ":" + messageID
}
