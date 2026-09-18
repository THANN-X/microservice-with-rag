// WHAT: Test suite สำหรับ cartQueryService
//
// จุดสำคัญที่สุดที่เทสชุดนี้ล็อกไว้คือ contract กับ frontend:
//   GetCart ต้องคืน "empty cart + nil error" เมื่อยังไม่มีตะกร้า — ไม่ใช่ 404
//   ถ้าใครมา refactor แล้วเปลี่ยนเป็น NewNotFoundError (ซึ่งดูสมเหตุสมผลกว่าสำหรับคนไม่รู้บริบท)
//   หน้าตะกร้าฝั่ง frontend จะพังทั้งระบบ
//
// การรัน test:
//   go test ./internal/core/service/query/... -v
package query_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"cart_service/internal/core/domain"
	"cart_service/internal/core/service/query"
	"errs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockCartQueryRepo คือ mock ของ repo.CartQueryRepository
// query side มี method เดียว → mock สั้นกว่า command side มาก
type mockCartQueryRepo struct {
	mock.Mock
}

func (m *mockCartQueryRepo) GetCartByUserID(ctx context.Context, userID uint) (*domain.Cart, error) {
	args := m.Called(ctx, userID)
	cart, _ := args.Get(0).(*domain.Cart)
	return cart, args.Error(1)
}

func assertInternalAppError(t *testing.T, err error) {
	t.Helper()
	appErr, ok := err.(errs.AppError)
	assert.True(t, ok, "expected errs.AppError, got %T", err)
	assert.Equal(t, http.StatusInternalServerError, appErr.Code)
}

// --- GetCart ---

// TestGetCart_Success ตรวจ mapper ด้วย: domain.CartItem → dto.CartItemRes ต้องครบทุก field
// WHY เทสถึง item level? — เทสฝั่ง command ใช้ cart ที่ Items ว่าง ทำให้ mapper ไม่เคยถูกตรวจจริง
func TestGetCart_Success(t *testing.T) {
	repo := new(mockCartQueryRepo)
	svc := query.NewCartQueryService(repo)
	ctx := context.Background()
	userID := uint(7)

	now := time.Now()
	cart := &domain.Cart{
		ID:     42,
		UserID: userID,
		Items: []domain.CartItem{
			{
				CartID:      42,
				VariantID:   5,
				Quantity:    3,
				ProductName: "เสื้อยืด",
				VariantName: "สีดำ / M",
				Price:       299.50,
				ImageURL:    "https://example.com/shirt.jpg",
				AddedAt:     now,
				UpdatedAt:   now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	repo.On("GetCartByUserID", ctx, userID).Return(cart, nil).Once()

	res, err := svc.GetCart(ctx, userID)

	// WHY require ไม่ใช่ assert? — require หยุดเทสทันทีเมื่อ fail
	// ถ้าใช้ assert แล้ว res เป็น nil บรรทัดถัดไปจะ panic → test binary ตายยกชุด เทสอื่นไม่ได้รัน
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, uint(42), res.CartID)
	assert.Equal(t, userID, res.UserID)

	require.Len(t, res.Items, 1)
	item := res.Items[0]
	assert.Equal(t, uint(5), item.VariantID)
	assert.Equal(t, 3, item.Quantity)
	assert.Equal(t, "เสื้อยืด", item.ProductName)
	assert.Equal(t, "สีดำ / M", item.VariantName)
	assert.Equal(t, 299.50, item.Price)
	assert.Equal(t, "https://example.com/shirt.jpg", item.ImageURL)

	repo.AssertExpectations(t)
}

// TestGetCart_NotFound_ReturnsEmptyCartNotError คือเทสที่สำคัญที่สุดในไฟล์นี้
// Lazy Creation philosophy: GET ต้องไม่ trigger side effect และต้องไม่คืน 404
func TestGetCart_NotFound_ReturnsEmptyCartNotError(t *testing.T) {
	repo := new(mockCartQueryRepo)
	svc := query.NewCartQueryService(repo)
	ctx := context.Background()
	userID := uint(7)

	repo.On("GetCartByUserID", ctx, userID).Return(nil, domain.ErrRecordNotFound).Once()

	res, err := svc.GetCart(ctx, userID)

	// ต้องไม่ error — ไม่ใช่ 404
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, userID, res.UserID)
	assert.Equal(t, uint(0), res.CartID, "ยังไม่มีตะกร้าจริง cart_id ต้องเป็น 0")

	// WHY assert.NotNil กับ slice ว่าง?
	//   - nil slice ถูก marshal เป็น JSON null, empty slice เป็น []
	//   - frontend ทำ cart.items.map(...) → ถ้าได้ null จะ crash ทันที
	assert.NotNil(t, res.Items, "Items ต้องเป็น empty slice ไม่ใช่ nil (JSON ต้องเป็น [] ไม่ใช่ null)")
	assert.Empty(t, res.Items)

	repo.AssertExpectations(t)
}

// TestGetCart_RepoError_ReturnsUnexpectedError — error อื่นที่ไม่ใช่ not-found ต้องเป็น 500
// (ห้ามกลืนเป็น empty cart เพราะจะซ่อนปัญหา DB จาก monitoring)
func TestGetCart_RepoError_ReturnsUnexpectedError(t *testing.T) {
	repo := new(mockCartQueryRepo)
	svc := query.NewCartQueryService(repo)
	ctx := context.Background()
	userID := uint(7)

	repo.On("GetCartByUserID", ctx, userID).Return(nil, errors.New("connection refused")).Once()

	res, err := svc.GetCart(ctx, userID)

	assert.Nil(t, res)
	assertInternalAppError(t, err)
	repo.AssertExpectations(t)
}

// TestGetCart_EmptyItems_ReturnsEmptySliceNotNil — ตะกร้ามีอยู่จริงแต่ไม่มีของ
// เคสนี้เกิดหลัง ClearCart หรือหลังลบ item ชิ้นสุดท้าย ต้องได้ [] เหมือนกัน
func TestGetCart_EmptyItems_ReturnsEmptySliceNotNil(t *testing.T) {
	repo := new(mockCartQueryRepo)
	svc := query.NewCartQueryService(repo)
	ctx := context.Background()
	userID := uint(7)

	cart := &domain.Cart{
		ID:        42,
		UserID:    userID,
		Items:     []domain.CartItem{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.On("GetCartByUserID", ctx, userID).Return(cart, nil).Once()

	res, err := svc.GetCart(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, res)
	assert.NotNil(t, res.Items)
	assert.Empty(t, res.Items)
	assert.Equal(t, uint(42), res.CartID, "ตะกร้ามีอยู่จริง cart_id ต้องไม่ใช่ 0")

	repo.AssertExpectations(t)
}
