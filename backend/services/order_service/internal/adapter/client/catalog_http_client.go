// WHAT: HTTP client สำหรับ catalog_service
// WHY REST แทน gRPC?
//   - catalog_service expose REST API อยู่แล้ว (GET /catalog/variants/:id)
//   - ไม่ต้องเพิ่ม .proto file + codegen สำหรับ 1 endpoint
//   - TODO: เปลี่ยนเป็น gRPC ถ้าต้องการ strict contract หรือ streaming
//
// WHY timeout 5 วินาที?
//   - PlaceOrder เป็น synchronous flow — ลูกค้ารอ response อยู่
//   - 5s เพียงพอสำหรับ internal service call ใน Docker network
//   - circuit-break ถ้าเกิน → คืน error ให้ลูกค้า (ดีกว่าแขวน request ค้างไว้)
package client

import (
	"context"
	"encoding/json"
	"errs"
	"fmt"
	"net/http"
	gateway "order_service/internal/core/port/gateway"
	"time"
)

type catalogHTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewCatalogClient สร้าง HTTP client ไปยัง catalog_service
// baseURL เช่น "http://catalog-service-app:3005"
func NewCatalogClient(baseURL string) gateway.CatalogClient {
	return &catalogHTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// variantInfoRes แมปกับ JSON response จาก GET /catalog/variants/:id
type variantInfoRes struct {
	VariantID   uint    `json:"variant_id"`
	ProductName string  `json:"product_name"`
	VariantName string  `json:"variant_name"`
	Price       float64 `json:"price"`
	ImageURL    string  `json:"image_url"`
}

// GetVariantSnapshot เรียก catalog_service เพื่อดึง price + product snapshot
// WHY ต้อง pass ctx?
//   - ถ้า request ต้นทาง cancel (timeout/disconnect) → HTTP call ยกเลิกตาม
//   - ป้องกัน goroutine leak และ wasted work บน catalog_service
func (c *catalogHTTPClient) GetVariantSnapshot(ctx context.Context, variantID uint) (*gateway.VariantSnapshot, error) {
	url := fmt.Sprintf("%s/catalog/variants/%d", c.baseURL, variantID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("catalog client: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog client: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errs.NewNotFoundError(fmt.Sprintf("variant %d not found in catalog", variantID))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog client: unexpected status %d for variant %d", resp.StatusCode, variantID)
	}

	var res variantInfoRes
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("catalog client: decode response: %w", err)
	}

	return &gateway.VariantSnapshot{
		VariantID:   res.VariantID,
		ProductName: res.ProductName,
		VariantName: res.VariantName,
		Price:       res.Price,
		ImageURL:    res.ImageURL,
	}, nil
}
