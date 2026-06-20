// WHAT: HTTP Handlers สำหรับ Order endpoints (Fiber framework)
// WHY แยก Handler ออกจาก Service?
//   - Handler รับผิดชอบ HTTP concerns เท่านั้น: parse request, validate, call service, format response
//   - Service ไม่รู้จัก HTTP → ทดสอบได้ง่าย (mock service)
//   - ถ้าเปลี่ยน framework → แก้แค่ handler ไม่กระทบ service/domain
package http

import (
	"errs"
	"httpcore"
	service "order_service/internal/core/port/service"
	dto "order_service/internal/core/port/service/dto"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type orderHandler struct {
	cmdService   service.OrderCommandService
	queryService service.OrderQueryService
	validator    *validator.Validate
}

func NewOrderHandler(cmd service.OrderCommandService, query service.OrderQueryService) *orderHandler {
	return &orderHandler{
		cmdService:   cmd,
		queryService: query,
		validator:    validator.New(),
	}
}

// ─── Command Handlers ─────────────────────────────────────────────────────────

// POST /orders
// PlaceOrder สร้าง Order ใหม่ (Customer เท่านั้น — ผ่าน AuthMiddleware)
func (h *orderHandler) PlaceOrder(c *fiber.Ctx) error {
	req := &dto.CreateOrderReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("invalid request body"))
	}
	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	// ดึง customer ID จาก JWT token (set ไว้โดย AuthMiddleware)
	customerID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("invalid user ID in token"))
	}

	orderRes, err := h.cmdService.PlaceOrder(c.UserContext(), customerID, req)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	// WHY 201 Created แทน 200?
	//   - 201 = resource ถูกสร้างสำเร็จ → client รู้ว่า order ถูก create ไม่ใช่แค่ OK
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "order placed successfully",
		"order":   orderRes,
	})
}

// POST /orders/:id/cancel
// CancelOrder ยกเลิก Order โดย customer (ตรวจ ownership ใน service)
func (h *orderHandler) CancelOrder(c *fiber.Ctx) error {
	orderID := c.Params("id")
	if orderID == "" {
		return httpcore.HandleError(c, errs.NewValidationError("order ID is required"))
	}

	req := &dto.CancelOrderReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("invalid request body"))
	}
	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	customerID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("invalid user ID in token"))
	}

	if err := h.cmdService.CancelOrder(c.UserContext(), orderID, customerID, req.Reason); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "order cancelled successfully",
	})
}

// POST /orders/admin/:id/cancel
// AdminCancelOrder ยกเลิก Order โดย admin (ไม่มี ownership check)
func (h *orderHandler) AdminCancelOrder(c *fiber.Ctx) error {
	orderID := c.Params("id")
	if orderID == "" {
		return httpcore.HandleError(c, errs.NewValidationError("order ID is required"))
	}

	req := &dto.CancelOrderReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("invalid request body"))
	}
	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	if err := h.cmdService.AdminCancelOrder(c.UserContext(), orderID, req.Reason); err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "order cancelled by admin",
	})
}

// ─── Query Handlers ───────────────────────────────────────────────────────────

// GET /orders/:id
// GetOrder ดึง Order เดี่ยว (ตรวจ ownership ใน service)
func (h *orderHandler) GetOrder(c *fiber.Ctx) error {
	orderID := c.Params("id")
	if orderID == "" {
		return httpcore.HandleError(c, errs.NewValidationError("order ID is required"))
	}

	customerID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("invalid user ID in token"))
	}

	orderRes, err := h.queryService.GetOrderByID(c.UserContext(), orderID, customerID)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(orderRes)
}

// NOTE: ListMyOrders / ListAllOrders / AdminGetOrder ถูกย้ายไป order_history_service (CQRS read side)
// BFF จะ route GET /orders, GET /orders/admin/all ไปที่ order_history_service แทน

// ─── Payment Handlers ─────────────────────────────────────────────────────────

// POST /orders/:id/pay
// ProcessPayment ลูกค้าชำระเงินสำหรับ Order ที่ stock confirmed แล้ว
func (h *orderHandler) ProcessPayment(c *fiber.Ctx) error {
	orderID := c.Params("id")
	if orderID == "" {
		return httpcore.HandleError(c, errs.NewValidationError("order ID is required"))
	}

	req := &dto.ProcessPaymentReq{}
	if err := c.BodyParser(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError("invalid request body"))
	}
	if err := h.validator.Struct(req); err != nil {
		return httpcore.HandleError(c, errs.NewValidationError(err.Error()))
	}

	// WHY ดึง customerID จาก JWT (Locals) แทนรับจาก request body?
	//   - ป้องกัน client ส่ง customerID เป็นของคนอื่น → จ่ายเงินแทนคนอื่นได้
	//   - JWT token ออกโดย auth_service → tamper-proof → trust ได้
	customerID, ok := c.Locals("user_id").(uint)
	if !ok {
		return httpcore.HandleError(c, errs.NewUnauthorizedError("invalid user ID in token"))
	}

	paymentRes, err := h.cmdService.ProcessPayment(c.UserContext(), orderID, customerID, req)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "payment processed successfully",
		"payment": paymentRes,
	})
}

// POST /webhook/payment
// HandlePaymentWebhook รับ callback จาก payment gateway (ไม่มี auth middleware)
//
// WHY ไม่มี auth middleware?
//   - endpoint นี้ถูกเรียกโดย gateway (Stripe/Omise) ไม่ใช่ user → ไม่มี JWT
//   - ป้องกันด้วย signature verification แทน (VerifyWebhook ใน service layer)
func (h *orderHandler) HandlePaymentWebhook(c *fiber.Ctx) error {
	req := &dto.PaymentWebhookReq{
		// WHY อ่าน signature จาก header?
		//   - Gateway จะ sign payload และใส่ signature ใน HTTP header (เช่น "X-Webhook-Signature")
		//   - Body เป็น raw bytes ที่ยังไม่ parse → ต้องส่งเป็น []byte ให้ VerifyWebhook ตรวจ HMAC
		Signature: c.Get("Stripe-Signature"),
		Payload:   c.Body(), // WHY c.Body() (ไม่ใช่ BodyParser)? — ต้องการ raw bytes สำหรับ HMAC verify
	}

	if err := h.cmdService.HandlePaymentWebhook(c.UserContext(), req); err != nil {
		return httpcore.HandleError(c, err)
	}

	// WHY return 200 OK ว่างเปล่า?
	//   - Gateway จะ retry ถ้าไม่ได้รับ 2xx → return OK บอก gateway ว่า "received, stop retrying"
	return c.SendStatus(fiber.StatusOK)
}
