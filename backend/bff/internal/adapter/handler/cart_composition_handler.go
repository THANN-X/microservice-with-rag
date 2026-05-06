package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

type cartCompositionHandler struct {
	cartBaseURL    string
	catalogBaseURL string
	httpClient     *http.Client
}

type composedCartRes struct {
	CartID    uint               `json:"cart_id"`
	UserID    uint               `json:"user_id"`
	Items     []composedCartItem `json:"items"`
	CreatedAt string             `json:"created_at"`
	UpdatedAt string             `json:"updated_at"`
}

type composedCartItem struct {
	VariantID   uint    `json:"variant_id"`
	Quantity    int     `json:"quantity"`
	ProductName string  `json:"product_name,omitempty"`
	VariantName string  `json:"variant_name,omitempty"`
	Price       float64 `json:"price,omitempty"`
	ImageURL    string  `json:"image_url,omitempty"`
	AddedAt     string  `json:"added_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type catalogVariantInfoRes struct {
	ProductName string   `json:"product_name"`
	VariantName string   `json:"variant_name"`
	Price       float64  `json:"price"`
	ImageURL    string   `json:"image_url"`
	ImageURLs   []string `json:"image_urls"`
}

func NewCartCompositionHandler(cartHost string, catalogHost string) *cartCompositionHandler {
	return &cartCompositionHandler{
		cartBaseURL:    "http://" + cartHost,
		catalogBaseURL: "http://" + catalogHost,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetCart returns cart data enriched via API composition (cart + catalog).
func (h *cartCompositionHandler) GetCart(c *fiber.Ctx) error {
	cart, statusCode, err := h.fetchCart(c)
	if err != nil {
		return c.Status(statusCode).JSON(fiber.Map{"error": err.Error()})
	}

	for i := range cart.Items {
		item := &cart.Items[i]
		if item.ProductName != "" && item.VariantName != "" && item.Price > 0 && item.ImageURL != "" {
			continue
		}

		variant, variantStatus, variantErr := h.fetchVariantInfo(c, item.VariantID)
		if variantErr != nil {
			// Ignore missing variant details and keep base cart data.
			if variantStatus == fiber.StatusNotFound {
				continue
			}
			continue
		}

		if item.ProductName == "" {
			item.ProductName = variant.ProductName
		}
		if item.VariantName == "" {
			item.VariantName = variant.VariantName
		}
		if item.Price == 0 {
			item.Price = variant.Price
		}
		if item.ImageURL == "" {
			if variant.ImageURL != "" {
				item.ImageURL = variant.ImageURL
			} else if len(variant.ImageURLs) > 0 {
				item.ImageURL = variant.ImageURLs[0]
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(cart)
}

func (h *cartCompositionHandler) fetchCart(c *fiber.Ctx) (*composedCartRes, int, error) {
	url := h.cartBaseURL + "/cart"
	req, err := http.NewRequestWithContext(c.UserContext(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fiber.StatusInternalServerError, fmt.Errorf("failed to create cart request")
	}
	if auth := c.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if userID, ok := c.Locals("user_id").(uint); ok {
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	}
	if role, ok := c.Locals("role").(string); ok {
		req.Header.Set("X-Role", role)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fiber.StatusBadGateway, fmt.Errorf("cart service unavailable")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fiber.StatusBadGateway, fmt.Errorf("failed to read cart response")
	}
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("cart service error")
	}

	res := &composedCartRes{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, res); err != nil {
			return nil, fiber.StatusBadGateway, fmt.Errorf("invalid cart response")
		}
	}
	if res.Items == nil {
		res.Items = []composedCartItem{}
	}

	return res, fiber.StatusOK, nil
}

func (h *cartCompositionHandler) fetchVariantInfo(c *fiber.Ctx, variantID uint) (*catalogVariantInfoRes, int, error) {
	url := fmt.Sprintf("%s/catalog/variants/%d", h.catalogBaseURL, variantID)
	req, err := http.NewRequestWithContext(c.UserContext(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fiber.StatusInternalServerError, fmt.Errorf("failed to create catalog request")
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fiber.StatusBadGateway, fmt.Errorf("catalog service unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("catalog service error")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fiber.StatusBadGateway, fmt.Errorf("failed to read catalog response")
	}

	res := &catalogVariantInfoRes{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, res); err != nil {
			return nil, fiber.StatusBadGateway, fmt.Errorf("invalid catalog response")
		}
	}
	return res, fiber.StatusOK, nil
}
