package http

import (
	serviceport "order_history_service/internal/core/port/service"
	"order_history_service/internal/core/port/service/dto"

	"github.com/gofiber/fiber/v2"
	"httpcore"
)

type orderHistoryHandler struct {
	queryService serviceport.OrderHistoryQueryService
}

func NewOrderHistoryHandler(queryService serviceport.OrderHistoryQueryService) *orderHistoryHandler {
	return &orderHistoryHandler{queryService: queryService}
}

// ListMyOrders godoc
// @Summary     List order history for authenticated user
// @Tags        order-history
// @Produce     json
// @Param       page   query int    false "Page number (default: 1)"
// @Param       limit  query int    false "Items per page (default: 10, max: 100)"
// @Param       status query string false "Filter by status (PENDING, CONFIRMED, CANCELLED)"
// @Success     200 {object} dto.OrderHistoryListRes
// @Router      /order-history [get]
func (h *orderHistoryHandler) ListMyOrders(c *fiber.Ctx) error {
	customerID, ok := c.Locals("user_id").(uint)
	if !ok || customerID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	req := &dto.ListOrderHistoryReq{}
	if err := c.QueryParser(req); err != nil {
		return httpcore.HandleError(c, err)
	}

	result, err := h.queryService.ListMyOrders(c.UserContext(), customerID, req)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.JSON(result)
}

// GetOrder godoc
// @Summary     Get a single order from history
// @Tags        order-history
// @Produce     json
// @Param       orderId path string true "Order ID (UUID)"
// @Success     200 {object} dto.OrderHistoryRes
// @Router      /order-history/{orderId} [get]
func (h *orderHistoryHandler) GetOrder(c *fiber.Ctx) error {
	customerID, ok := c.Locals("user_id").(uint)
	if !ok || customerID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	orderID := c.Params("orderId")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "order id is required"})
	}

	result, err := h.queryService.GetOrderByID(c.UserContext(), orderID, customerID)
	if err != nil {
		return httpcore.HandleError(c, err)
	}

	return c.JSON(result)
}
