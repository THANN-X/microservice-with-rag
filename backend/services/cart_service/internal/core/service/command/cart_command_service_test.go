package command_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"cart_service/internal/core/domain"
	"cart_service/internal/core/port/service/dto"
	"cart_service/internal/core/service/command"
	"errs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockCartCommandRepo คือ mock implementation ของ repo.CartCommandRepository (testify/mock)
// ใช้ตรวจว่า cartCommandService เรียก repo ด้วย method/argument ที่ถูกต้อง โดยไม่ต้องต่อ Postgres จริง
type mockCartCommandRepo struct {
	mock.Mock
}

func (m *mockCartCommandRepo) FindOrCreateByUserID(ctx context.Context, userID uint) (*domain.Cart, error) {
	args := m.Called(ctx, userID)
	cart, _ := args.Get(0).(*domain.Cart)
	return cart, args.Error(1)
}

func (m *mockCartCommandRepo) UpsertItem(ctx context.Context, cartID uint, variantID uint, quantity int, meta domain.CartItemMeta) error {
	args := m.Called(ctx, cartID, variantID, quantity, meta)
	return args.Error(0)
}

func (m *mockCartCommandRepo) SetItemQuantity(ctx context.Context, cartID uint, variantID uint, quantity int) error {
	args := m.Called(ctx, cartID, variantID, quantity)
	return args.Error(0)
}

func (m *mockCartCommandRepo) RemoveItem(ctx context.Context, cartID uint, variantID uint) error {
	args := m.Called(ctx, cartID, variantID)
	return args.Error(0)
}

func (m *mockCartCommandRepo) ClearCart(ctx context.Context, cartID uint) error {
	args := m.Called(ctx, cartID)
	return args.Error(0)
}

func sampleCart(userID uint) *domain.Cart {
	return &domain.Cart{
		ID:        1,
		UserID:    userID,
		Items:     []domain.CartItem{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func assertNotFoundAppError(t *testing.T, err error) {
	t.Helper()
	appErr, ok := err.(errs.AppError)
	assert.True(t, ok, "expected errs.AppError, got %T", err)
	assert.Equal(t, http.StatusNotFound, appErr.Code)
}

// assertInternalAppError ตรวจ status code 500 ไม่ใช่แค่ assert.Error()
// WHY? — assert.Error() ผ่านทั้ง 404 และ 500 จับไม่ได้ถ้า error mapping สลับกัน
func assertInternalAppError(t *testing.T, err error) {
	t.Helper()
	appErr, ok := err.(errs.AppError)
	assert.True(t, ok, "expected errs.AppError, got %T", err)
	assert.Equal(t, http.StatusInternalServerError, appErr.Code)
}

// --- AddItem ---

func TestAddItem_Success(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)

	req := &dto.AddCartItemReq{
		VariantID:   10,
		Quantity:    2,
		ProductName: "T-Shirt",
		Price:       299.0,
	}
	wantMeta := domain.CartItemMeta{
		ProductName: req.ProductName,
		VariantName: req.VariantName,
		Price:       req.Price,
		ImageURL:    req.ImageURL,
	}

	// เรียก FindOrCreateByUserID 2 ครั้ง: ครั้งแรกเพื่อเอา cart.ID, ครั้งสองเพื่อ re-fetch state ล่าสุด
	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Twice()
	repo.On("UpsertItem", ctx, cart.ID, req.VariantID, req.Quantity, wantMeta).Return(nil).Once()

	res, err := svc.AddItem(ctx, userID, req)

	assert.NoError(t, err)
	assert.Equal(t, cart.ID, res.CartID)
	assert.Equal(t, userID, res.UserID)
	repo.AssertExpectations(t)
}

func TestAddItem_FindOrCreateFails_ReturnsUnexpectedError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)

	repo.On("FindOrCreateByUserID", ctx, userID).Return(nil, errors.New("db down")).Once()

	res, err := svc.AddItem(ctx, userID, &dto.AddCartItemReq{VariantID: 1, Quantity: 1})

	assert.Nil(t, res)
	assert.Error(t, err)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "UpsertItem", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAddItem_UpsertFails_ReturnsUnexpectedError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)
	req := &dto.AddCartItemReq{VariantID: 10, Quantity: 2}

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Once()
	repo.On("UpsertItem", ctx, cart.ID, req.VariantID, req.Quantity, mock.Anything).Return(errors.New("insert failed")).Once()

	res, err := svc.AddItem(ctx, userID, req)

	assert.Nil(t, res)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

// --- RemoveItem ---

func TestRemoveItem_Success(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	variantID := uint(99)
	cart := sampleCart(userID)

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Twice()
	repo.On("RemoveItem", ctx, cart.ID, variantID).Return(nil).Once()

	res, err := svc.RemoveItem(ctx, userID, variantID)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	repo.AssertExpectations(t)
}

func TestRemoveItem_NotFound_ReturnsNotFoundAppError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	variantID := uint(99)
	cart := sampleCart(userID)

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Once()
	repo.On("RemoveItem", ctx, cart.ID, variantID).Return(domain.ErrItemNotFound).Once()

	res, err := svc.RemoveItem(ctx, userID, variantID)

	assert.Nil(t, res)
	assertNotFoundAppError(t, err)
	repo.AssertExpectations(t)
}

func TestRemoveItem_RepoError_ReturnsUnexpectedError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	variantID := uint(99)
	cart := sampleCart(userID)

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Once()
	repo.On("RemoveItem", ctx, cart.ID, variantID).Return(errors.New("db error")).Once()

	res, err := svc.RemoveItem(ctx, userID, variantID)

	assert.Nil(t, res)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

// --- UpdateItemQuantity ---

func TestUpdateItemQuantity_ZeroQuantity_RemovesItem(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)
	req := &dto.UpdateCartItemReq{VariantID: 5, Quantity: 0}

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Twice()
	repo.On("RemoveItem", ctx, cart.ID, req.VariantID).Return(nil).Once()

	res, err := svc.UpdateItemQuantity(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "SetItemQuantity", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateItemQuantity_PositiveQuantity_SetsQuantity(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)
	req := &dto.UpdateCartItemReq{VariantID: 5, Quantity: 3}

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Twice()
	repo.On("SetItemQuantity", ctx, cart.ID, req.VariantID, req.Quantity).Return(nil).Once()

	res, err := svc.UpdateItemQuantity(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "RemoveItem", mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateItemQuantity_NotFound_ReturnsNotFoundAppError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)
	req := &dto.UpdateCartItemReq{VariantID: 5, Quantity: 3}

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Once()
	repo.On("SetItemQuantity", ctx, cart.ID, req.VariantID, req.Quantity).Return(domain.ErrItemNotFound).Once()

	res, err := svc.UpdateItemQuantity(ctx, userID, req)

	assert.Nil(t, res)
	assertNotFoundAppError(t, err)
	repo.AssertExpectations(t)
}

// TestUpdateItemQuantity_ZeroQuantity_NotFound_ReturnsNotFoundAppError
// คุม branch qty<=0 → RemoveItem คืน ErrItemNotFound ต้อง map เป็น 404 ไม่ใช่ 500
// (branch นี้คู่ขนานกับ SetItemQuantity แต่เดิมไม่มีเทสคุม)
func TestUpdateItemQuantity_ZeroQuantity_NotFound_ReturnsNotFoundAppError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)
	req := &dto.UpdateCartItemReq{VariantID: 5, Quantity: 0}

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Once()
	repo.On("RemoveItem", ctx, cart.ID, req.VariantID).Return(domain.ErrItemNotFound).Once()

	res, err := svc.UpdateItemQuantity(ctx, userID, req)

	assert.Nil(t, res)
	assertNotFoundAppError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_ZeroQuantity_RepoError_ReturnsUnexpectedError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)
	req := &dto.UpdateCartItemReq{VariantID: 5, Quantity: 0}

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Once()
	repo.On("RemoveItem", ctx, cart.ID, req.VariantID).Return(errors.New("db error")).Once()

	res, err := svc.UpdateItemQuantity(ctx, userID, req)

	assert.Nil(t, res)
	assertInternalAppError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_SetQuantityRepoError_ReturnsUnexpectedError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)
	req := &dto.UpdateCartItemReq{VariantID: 5, Quantity: 3}

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Once()
	repo.On("SetItemQuantity", ctx, cart.ID, req.VariantID, req.Quantity).Return(errors.New("db error")).Once()

	res, err := svc.UpdateItemQuantity(ctx, userID, req)

	assert.Nil(t, res)
	assertInternalAppError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateItemQuantity_FindOrCreateFails_ReturnsUnexpectedError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)

	repo.On("FindOrCreateByUserID", ctx, userID).Return(nil, errors.New("db down")).Once()

	res, err := svc.UpdateItemQuantity(ctx, userID, &dto.UpdateCartItemReq{VariantID: 5, Quantity: 3})

	assert.Nil(t, res)
	assertInternalAppError(t, err)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "SetItemQuantity", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "RemoveItem", mock.Anything, mock.Anything, mock.Anything)
}

// --- ClearCart ---

func TestClearCart_Success(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Once()
	repo.On("ClearCart", ctx, cart.ID).Return(nil).Once()

	err := svc.ClearCart(ctx, userID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestClearCart_RepoError_ReturnsUnexpectedError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)
	cart := sampleCart(userID)

	repo.On("FindOrCreateByUserID", ctx, userID).Return(cart, nil).Once()
	repo.On("ClearCart", ctx, cart.ID).Return(errors.New("db error")).Once()

	err := svc.ClearCart(ctx, userID)

	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestClearCart_FindOrCreateFails_ReturnsUnexpectedError(t *testing.T) {
	repo := new(mockCartCommandRepo)
	svc := command.NewCartCommandService(repo)
	ctx := context.Background()
	userID := uint(1)

	repo.On("FindOrCreateByUserID", ctx, userID).Return(nil, errors.New("db down")).Once()

	err := svc.ClearCart(ctx, userID)

	assert.Error(t, err)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "ClearCart", mock.Anything, mock.Anything)
}
