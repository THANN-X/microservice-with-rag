package mongo

import (
	"context"
	"order_history_service/internal/core/domain"
	repo "order_history_service/internal/core/port/repo"
	"time"

	"errs"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const orderHistoryCollection = "order_history"

type orderHistoryDocument struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	OrderID         string             `bson:"order_id"`
	CustomerID      uint               `bson:"customer_id"`
	Status          string             `bson:"status"`
	TotalAmount     float64            `bson:"total_amount"`
	Items           []itemDoc          `bson:"items"`
	ShippingAddress addressDoc         `bson:"shipping_address"`
	Note            string             `bson:"note"`
	CancelReason    string             `bson:"cancel_reason"`
	CreatedAt       time.Time          `bson:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at"`
}

type itemDoc struct {
	VariantID uint    `bson:"variant_id"`
	Quantity  int     `bson:"quantity"`
	UnitPrice float64 `bson:"unit_price"`
}

type addressDoc struct {
	FullName    string `bson:"full_name"`
	Phone       string `bson:"phone"`
	AddressLine string `bson:"address_line"`
	SubDistrict string `bson:"sub_district"`
	District    string `bson:"district"`
	Province    string `bson:"province"`
	PostalCode  string `bson:"postal_code"`
}

type orderHistoryRepository struct {
	col *mongo.Collection
}

func NewOrderHistoryRepository(db *mongo.Database) (repo.OrderHistoryWriteRepository, repo.OrderHistoryReadRepository) {
	r := &orderHistoryRepository{col: db.Collection(orderHistoryCollection)}
	return r, r
}

// EnsureIndexes สร้าง indexes ที่จำเป็น (idempotent)
func EnsureIndexes(db *mongo.Database) error {
	col := db.Collection(orderHistoryCollection)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "order_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{
				{Key: "customer_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
	}

	_, err := col.Indexes().CreateMany(ctx, indexes)
	return err
}

// --- Write Methods ---

func (r *orderHistoryRepository) Upsert(ctx context.Context, order *domain.OrderHistory) error {
	now := time.Now()

	items := toItemDocs(order.Items)
	addr := toAddressDoc(order.ShippingAddress)

	filter := bson.M{"order_id": order.OrderID}
	update := bson.M{
		"$set": bson.M{
			"customer_id":      order.CustomerID,
			"status":           order.Status,
			"total_amount":     order.TotalAmount,
			"items":            items,
			"shipping_address": addr,
			"note":             order.Note,
			"cancel_reason":    order.CancelReason,
			"updated_at":       now,
		},
		"$setOnInsert": bson.M{
			"created_at": order.CreatedAt,
		},
	}

	_, err := r.col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func (r *orderHistoryRepository) UpdateStatus(ctx context.Context, orderID string, status string) error {
	filter := bson.M{"order_id": orderID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errs.NewNotFoundError("order not found in history")
	}
	return nil
}

func (r *orderHistoryRepository) MarkCancelled(ctx context.Context, orderID string, reason string) error {
	filter := bson.M{"order_id": orderID}
	update := bson.M{
		"$set": bson.M{
			"status":        "CANCELLED",
			"cancel_reason": reason,
			"updated_at":    time.Now(),
		},
	}

	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errs.NewNotFoundError("order not found in history")
	}
	return nil
}

// --- Read Methods ---

func (r *orderHistoryRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.OrderHistory, error) {
	filter := bson.M{"order_id": orderID}

	var doc orderHistoryDocument
	if err := r.col.FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errs.NewNotFoundError("order not found")
		}
		return nil, err
	}

	return toDomain(&doc), nil
}

func (r *orderHistoryRepository) FindByCustomerID(ctx context.Context, filter domain.OrderHistoryFilter) ([]domain.OrderHistory, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}

	query := bson.M{"customer_id": filter.CustomerID}
	if filter.Status != "" {
		query["status"] = filter.Status
	}

	total, err := r.col.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	findOpts := options.Find().SetSkip(int64((page - 1) * limit)).SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.col.Find(ctx, query, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var docs []orderHistoryDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, 0, err
	}

	orders := make([]domain.OrderHistory, len(docs))
	for i, d := range docs {
		orders[i] = *toDomain(&d)
	}

	return orders, total, nil
}

// --- Internal converters ---

func toItemDocs(items []domain.OrderHistoryItem) []itemDoc {
	docs := make([]itemDoc, len(items))
	for i, item := range items {
		docs[i] = itemDoc{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}
	return docs
}

func toAddressDoc(addr domain.ShippingAddress) addressDoc {
	return addressDoc{
		FullName:    addr.FullName,
		Phone:       addr.Phone,
		AddressLine: addr.AddressLine,
		SubDistrict: addr.SubDistrict,
		District:    addr.District,
		Province:    addr.Province,
		PostalCode:  addr.PostalCode,
	}
}

func toDomain(doc *orderHistoryDocument) *domain.OrderHistory {
	items := make([]domain.OrderHistoryItem, len(doc.Items))
	for i, item := range doc.Items {
		items[i] = domain.OrderHistoryItem{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}

	return &domain.OrderHistory{
		ID:          doc.ID.Hex(),
		OrderID:     doc.OrderID,
		CustomerID:  doc.CustomerID,
		Status:      doc.Status,
		TotalAmount: doc.TotalAmount,
		Items:       items,
		ShippingAddress: domain.ShippingAddress{
			FullName:    doc.ShippingAddress.FullName,
			Phone:       doc.ShippingAddress.Phone,
			AddressLine: doc.ShippingAddress.AddressLine,
			SubDistrict: doc.ShippingAddress.SubDistrict,
			District:    doc.ShippingAddress.District,
			Province:    doc.ShippingAddress.Province,
			PostalCode:  doc.ShippingAddress.PostalCode,
		},
		Note:         doc.Note,
		CancelReason: doc.CancelReason,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}
}
