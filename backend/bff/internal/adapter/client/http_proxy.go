package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

// What: ServiceProxy คือ reverse proxy สำหรับ forward HTTP request ไปยัง backend microservice
// Why:  BFF (Backend For Frontend) ทำหน้าที่เป็น API Gateway
//   - Frontend เรียกแค่ BFF endpoint เดียว → ไม่ต้องรู้จัก internal service แต่ละตัว
//   - ลดความซับซ้อนฝั่ง frontend, จัดการ CORS ที่จุดเดียว, ซ่อน internal network topology
//   - เวลาเพิ่ม service ใหม่ → แค่ register route ใน BFF ไม่ต้องแก้ frontend config
//
// TODO: เพิ่ม circuit breaker (เช่น sony/gobreaker) เพื่อ handle กรณี backend ล่ม
// TODO: เพิ่ม request/response logging สำหรับ distributed tracing (OpenTelemetry)
// TODO: เพิ่ม retry mechanism สำหรับ transient failures (เช่น 503)
type ServiceProxy struct {
	baseURL string
	client  *http.Client
}

// What: สร้าง ServiceProxy ใหม่ โดยรับ host:port ของ backend service
// Why:  แยก host ออกเป็น config (env var) เพื่อเปลี่ยนได้ตาม environment
//   - Local dev:   localhost:3001
//   - Docker:      auth-service-app:3001
//   - Production:  auth-service.internal:3001
func NewServiceProxy(host string) *ServiceProxy {
	return &ServiceProxy{
		baseURL: "http://" + host,
		client: &http.Client{
			// What: timeout 30 วินาที ป้องกัน request ค้างไม่มีกำหนด
			// Why:  ถ้า backend ไม่ตอบภายใน 30s → ตัดและคืน error ให้ frontend ดีกว่ารอไปเรื่อยๆ
			Timeout: 30 * time.Second,
		},
	}
}

// What: Forward รับ Fiber context + target path → proxy request ไปยัง backend service
// Why:  generic method ใช้ได้กับทุก endpoint ไม่ต้องเขียน typed client แยกทุก route
//
// Flow:
//
//	Frontend → [POST /api/auth/auth/user-login] → BFF
//	BFF strip prefix "/api/auth" → targetPath = "/auth/user-login"
//	BFF forward → http://auth-service:3001/auth/user-login
//	Backend response → BFF → Frontend (status code + body คงเดิม)
func (p *ServiceProxy) Forward(c *fiber.Ctx, targetPath string) error {
	// What: ประกอบ target URL จาก base + path + query string
	// Why:  query string ต้องส่งต่อเพื่อรองรับ pagination, filter, search
	//       เช่น /api/catalog/products?page=1&limit=10&search=shoes
	targetURL := p.baseURL + targetPath
	if q := string(c.Request().URI().QueryString()); q != "" {
		targetURL += "?" + q
	}

	// What: สร้าง upstream HTTP request — ใช้ method + body เดียวกับที่ frontend ส่งมา
	// Why:  BFF เป็น transparent proxy → ไม่เปลี่ยนแปลง request, แค่ route ไปถูกที่
	req, err := http.NewRequestWithContext(
		c.UserContext(),
		c.Method(),
		targetURL,
		bytes.NewReader(c.Body()),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create upstream request",
		})
	}

	// What: Forward headers ที่จำเป็นไปยัง backend
	// Why:  - Content-Type: backend ต้องรู้ว่า body เป็น JSON/form/etc. เพื่อ parse ถูก
	//       - Authorization: forward ไปให้ auth-service สำหรับ refresh-token / logout
	//       - X-Admin-Secret: ใช้สำหรับ admin registration endpoint โดยเฉพาะ
	//       - Cookie: forward refresh_token (HttpOnly cookie) ไปยัง auth-service
	//                 เพื่อรองรับ Approach B — refresh_token อยู่ใน HttpOnly cookie
	//                 Next.js rewrite proxy ส่ง Cookie header มาให้ BFF แล้ว BFF ต้อง forward ต่อไป
	if ct := string(c.Request().Header.ContentType()); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if auth := c.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if secret := c.Get("X-Admin-Secret"); secret != "" {
		req.Header.Set("X-Admin-Secret", secret)
	}
	if cookie := c.Get("Cookie"); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	// What: Forward user identity ที่ BFF validate แล้วไปยัง backend
	// Why:  BFF ทำ JWT validation แล้ว → services ไม่ต้อง validate เอง
	//       แค่อ่าน X-User-ID/X-Role headers ที่ BFF ส่งมา
	if userID, ok := c.Locals("user_id").(uint); ok {
		req.Header.Set("X-User-ID", fmt.Sprintf("%d", userID))
	}
	if role, ok := c.Locals("role").(string); ok {
		req.Header.Set("X-Role", role)
	}

	// What: ยิง request ไปยัง backend service
	resp, err := p.client.Do(req)
	if err != nil {
		// Why: 502 Bad Gateway = BFF ติดต่อ backend ไม่ได้
		//      แยกจาก 500 (BFF error เอง) เพื่อให้ frontend รู้ว่าปัญหาอยู่ upstream
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "upstream service unavailable",
		})
	}
	defer resp.Body.Close()

	// What: อ่าน response body จาก backend → ส่งกลับ frontend ทั้ง status code + body
	// Why:  BFF ไม่แปลง response — ส่งต่อตรงๆ เพื่อให้ frontend ได้ response เดียวกับที่ backend ส่งมา
	// TODO: จำกัดขนาด response body (เช่น 10MB) เพื่อป้องกัน memory overflow
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": "failed to read upstream response",
		})
	}

	// What: forward response headers จาก backend กลับไปยัง client
	// Why:  - Content-Type: client ต้อง parse response body ให้ถูกต้อง
	//       - Set-Cookie: forward HttpOnly refresh_token cookie ที่ auth-service set
	//                     ไปยัง Next.js proxy เพื่อให้ browser เก็บ cookie ที่ localhost:3000
	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	for _, sc := range resp.Header["Set-Cookie"] {
		c.Response().Header.Add("Set-Cookie", sc)
	}
	return c.Status(resp.StatusCode).Send(body)
}
