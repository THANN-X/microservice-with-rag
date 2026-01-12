package handler

import (
	"bff/internal/core/service"

	"github.com/gofiber/fiber/v2"
)

type BFFHandler struct {
	// Handler fields and methods
	svc *service.ProfileService
}

func NewBFFHandler(svc *service.ProfileService) *BFFHandler {
	return &BFFHandler{svc: svc}
}

func (h *BFFHandler) GetUserProfile(c *fiber.Ctx) error {
	id := c.Params("id")

	// Call the service to get user profile
	res, err := h.svc.GetUserProfile(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(res)
}
