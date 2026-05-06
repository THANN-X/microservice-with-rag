package main

import (
	"bff/internal/adapter/client"
	"bff/internal/adapter/handler"
	"bff/internal/core/service"
	bffmiddleware "bff/internal/middleware"
	"jwtutils"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// What: BFF (Backend For Frontend) — จุดเข้าเดียวของ frontend → backend ทั้งระบบ
//
// Why:  ระบบเป็น Microservices (auth, product, cart, order, catalog, order-history, AI)
//
//	frontend ไม่ควรเรียกแต่ละ service ตรงๆ เพราะ:
//	  1. ต้องจัดการ CORS หลายจุด
//	  2. frontend ต้องรู้ internal host/port ของทุก service
//	  3. ไม่สามารถ aggregate data หรือทำ protocol translation ได้
//
//	BFF แก้ปัญหาทั้งหมด:
//	  - REST Proxy: forward HTTP request ไปยัง backend service ที่ถูกต้อง
//	  - gRPC Bridge: แปลง REST (JSON) ↔ gRPC (Protobuf) สำหรับ AI service
//	  - Single CORS: frontend เรียกแค่ BFF ไม่ต้อง whitelist หลาย origin
//
// Architecture:
//
//	Frontend ──HTTP──► BFF (:8080) ──REST──► Auth (:3001)
//	                                ──REST──► Product (:3002)
//	                                ──REST──► Order (:3003)
//	                                ──REST──► Cart (:3004)
//	                                ──REST──► Catalog (:3005)
//	                                ──REST──► OrderHistory (:3006)
//	                                ──gRPC──► AI Service (:50051)
//
// TODO: เพิ่ม graceful shutdown (trap SIGINT/SIGTERM) เหมือน service อื่นๆ
// TODO: เพิ่ม health check endpoint (GET /health) สำหรับ Docker/K8s readiness probe
// TODO: เพิ่ม request logging middleware สำหรับ debugging
func main() {
	// ═══════════════════════════════════════════════════════════════════════════
	// 1) Load Service URLs จาก Environment Variables
	// ═══════════════════════════════════════════════════════════════════════════
	// What: อ่าน URL ของแต่ละ backend service จาก env (ถ้าไม่มีใช้ค่า default สำหรับ local dev)
	// Why:  URL เปลี่ยนตาม environment
	//         Local dev → localhost:3001
	//         Docker    → auth-service-app:3001 (ใช้ container name เป็น hostname)
	//         K8s       → auth-service.default.svc.cluster.local:3001
	authURL := getEnv("AUTH_SERVICE_URL", "localhost:3001")
	productURL := getEnv("PRODUCT_SERVICE_URL", "localhost:3002")
	orderURL := getEnv("ORDER_SERVICE_URL", "localhost:3003")
	cartURL := getEnv("CART_SERVICE_URL", "localhost:3004")
	catalogURL := getEnv("CATALOG_SERVICE_URL", "localhost:3005")
	orderHistoryURL := getEnv("ORDER_HISTORY_SERVICE_URL", "localhost:3006")
	aiServiceURL := getEnv("AI_SERVICE_URL", "localhost:50051")

	// ═══════════════════════════════════════════════════════════════════════════
	// 2) สร้าง REST Proxy Clients (HTTP reverse proxy)
	// ═══════════════════════════════════════════════════════════════════════════
	// What: ServiceProxy เป็น generic HTTP proxy — forward request ไปยัง backend service
	// Why:  ใช้ generic proxy แทน typed client เพราะ BFF แค่ route ไม่ได้แปลง data
	//       ลด boilerplate ไม่ต้องเขียน struct/interface สำหรับทุก endpoint ของทุก service
	authProxy := client.NewServiceProxy(authURL)
	productProxy := client.NewServiceProxy(productURL)
	orderProxy := client.NewServiceProxy(orderURL)
	cartProxy := client.NewServiceProxy(cartURL)
	catalogProxy := client.NewServiceProxy(catalogURL)
	orderHistoryProxy := client.NewServiceProxy(orderHistoryURL)

	// ═══════════════════════════════════════════════════════════════════════════
	// 3) สร้าง REST Proxy Handlers
	// ═══════════════════════════════════════════════════════════════════════════
	// What: ProxyHandler ตัด URL prefix ออก → forward ไป backend ที่กำหนด
	// Why:  Frontend เรียก "/api/auth/users/me" แต่ auth-service ใช้ "/users/me"
	//       ProxyHandler strip "/api/auth" ออกให้อัตโนมัติ
	//
	// Route Mapping (Frontend → Backend):
	//   /api/auth/*          → auth-service:3001/*              (strip "/api/auth")
	//   /api/products/*      → product-service:3002/products/*  (strip "/api")
	//   /api/categories/*    → product-service:3002/categories/* (strip "/api")
	//   /api/attributes/*    → product-service:3002/attributes/* (strip "/api")
	//   /api/cart/*          → cart-service:3004/cart/*          (strip "/api")
	//   /api/orders/*        → order-service:3003/orders/*      (strip "/api")
	//   /api/catalog/*       → catalog-service:3005/catalog/*   (strip "/api")
	//   /api/order-history/* → order-history:3006/order-history/*(strip "/api")
	authHandler := handler.NewProxyHandler(authProxy, "/api/auth")
	productHandler := handler.NewProxyHandler(productProxy, "/api")
	cartHandler := handler.NewProxyHandler(cartProxy, "/api")
	cartCompositionHandler := handler.NewCartCompositionHandler(cartURL, catalogURL)
	orderHandler := handler.NewProxyHandler(orderProxy, "/api")
	catalogHandler := handler.NewProxyHandler(catalogProxy, "/api")
	orderHistoryHandler := handler.NewProxyHandler(orderHistoryProxy, "/api")

	// ═══════════════════════════════════════════════════════════════════════════
	// 4) สร้าง gRPC Client สำหรับ AI Service (Protobuf)
	// ═══════════════════════════════════════════════════════════════════════════
	// What: ใช้ gRPC (protobuf) เรียก AI service แทน REST
	// Why:  gRPC เหมาะกับ internal service-to-service communication มากกว่า REST เพราะ:
	//         - Binary serialization (protobuf) เล็กกว่า JSON → เร็วกว่า
	//         - HTTP/2 multiplexing → ไม่ต้อง open connection ใหม่ทุก request
	//         - Strongly typed contract จาก .proto file → compile-time safety
	//         - เหมาะกับ AI service ที่ response อาจใหญ่ (product recommendations)
	//
	// How:  BFF ทำหน้าที่เป็น "Protocol Bridge"
	//         Frontend → [REST/JSON] → BFF → [gRPC/Protobuf] → AI Service
	//         AI Service → [gRPC/Protobuf] → BFF → [REST/JSON] → Frontend
	aiClient, err := client.NewAIGRPCClient(aiServiceURL)
	if err != nil {
		log.Printf("WARNING: ไม่สามารถเชื่อมต่อ AI service ได้: %v", err)
	}

	var chatHandler *handler.ChatHandler
	if aiClient != nil {
		chatService := service.NewChatService(aiClient)
		chatHandler = handler.NewChatHandler(chatService)
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// 5) JWT + Security Middleware
	// ═══════════════════════════════════════════════════════════════════════════
	jwtSecret := getEnv("JWT_SECRET", "my-secret-key-change-me")
	jwtService := jwtutils.NewJWTService(jwtSecret, "ecommerce_app")

	app := fiber.New()

	// What: CORS middleware — อนุญาต cross-origin requests จาก frontend
	// Why:  Frontend (localhost:3000) กับ BFF (localhost:8080) คนละ origin
	//       ถ้าไม่มี CORS → browser block request ทันที (Same-Origin Policy)
	app.Use(cors.New(cors.Config{
		AllowOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Admin-Secret",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))

	// What: Rate Limiter — จำกัดจำนวน request ต่อนาที
	// Why:  ป้องกัน brute force / DDoS — centralize ที่ BFF แทนที่จะใส่ทุก service
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
	}))

	// What: Gateway Auth Middleware — centralized JWT validation + admin guard
	// Why:  ทุก service ไม่ต้อง validate JWT เอง — BFF ทำที่เดียว
	//       แล้ว forward X-User-ID / X-Role headers ไปยัง backend
	app.Use(bffmiddleware.GatewayAuth(jwtService))

	// ═══════════════════════════════════════════════════════════════════════════
	// 6) Route Registration
	// ═══════════════════════════════════════════════════════════════════════════
	//
	// Why ใช้ app.All() + wildcard?
	//   BFF เป็น reverse proxy → ทุก HTTP method (GET/POST/PUT/DELETE/PATCH)
	//   ต้อง forward ไปยัง backend ได้หมด โดยไม่ต้องประกาศ route แยกทุก endpoint
	//
	// Why ต้องมีทั้ง "/api/xxx" (exact) และ "/api/xxx/*" (wildcard)?
	//   Fiber wildcard "*" match เฉพาะ path ที่มี segment หลัง slash
	//   "/api/products/*" match "/api/products/123" แต่ไม่ match "/api/products"
	//   ต้องประกาศ exact path แยกสำหรับ root endpoint (เช่น GET /api/products → list all)
	//
	// Why จัดเรียงจาก specific → generic?
	//   Fiber ใช้ first-match routing → route ที่ specific กว่าต้องอยู่ก่อน
	//   /chat ต้องอยู่ก่อน wildcard เพื่อไม่ให้ถูก catch ก่อน

	// --- AI Chat (gRPC → Protobuf) ---
	if chatHandler != nil {
		app.Post("/chat", chatHandler.Chat)
	} else {
		app.Post("/chat", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "AI service is not available",
			})
		})
	}

	// --- Auth Service (REST Proxy) ---
	// What: forward ทุก request ที่ขึ้นต้นด้วย /api/auth ไปยัง auth-service (:3001)
	// Why:  auth-service จัดการ: register, login, logout, refresh token, user profile
	// ตัวอย่าง:
	//   POST /api/auth/users/register      → auth:3001/users/register (public)
	//   POST /api/auth/auth/user-login     → auth:3001/auth/user-login (public)
	//   POST /api/auth/auth/admin-login    → auth:3001/auth/admin-login (public)
	//   POST /api/auth/auth/logout         → auth:3001/auth/logout
	//   POST /api/auth/auth/refresh-token  → auth:3001/auth/refresh-token
	//   GET  /api/auth/users/me            → auth:3001/users/me (protected)
	//   GET  /api/auth/users/:id           → auth:3001/users/:id (protected)
	app.All("/api/auth/*", authHandler.Handle)

	// --- Product Service (REST Proxy) ---
	// What: forward product/category/attribute requests ไปยัง product-service (:3002)
	// Why:  ทั้ง 3 resource อยู่ใน product-service เดียวกัน แค่คนละ route group
	//       ใช้ productProxy (เชื่อมไป product-service) ร่วมกัน
	// ตัวอย่าง:
	//   GET  /api/products                 → product:3002/products (public)
	//   GET  /api/products/:id             → product:3002/products/:id (public)
	//   POST /api/products/admin           → product:3002/products/admin (admin only)
	//   GET  /api/categories               → product:3002/categories (public)
	//   GET  /api/attributes               → product:3002/attributes (public)
	app.All("/api/products", productHandler.Handle)
	app.All("/api/products/*", productHandler.Handle)
	app.All("/api/categories", productHandler.Handle)
	app.All("/api/categories/*", productHandler.Handle)
	app.All("/api/attributes", productHandler.Handle)
	app.All("/api/attributes/*", productHandler.Handle)

	// --- Cart Service (REST Proxy) ---
	// What: forward cart requests ไปยัง cart-service (:3004)
	// Why:  cart-service จัดการตะกร้าสินค้า — ทุก endpoint ต้อง login (JWT required)
	//       BFF ไม่ validate JWT เอง → forward Authorization header ไปให้ cart-service validate
	// ตัวอย่าง:
	//   GET    /api/cart              → cart:3004/cart (ดูตะกร้า)
	//   POST   /api/cart/items        → cart:3004/cart/items (เพิ่มสินค้า)
	//   PUT    /api/cart/items/:id    → cart:3004/cart/items/:id (เปลี่ยนจำนวน)
	//   DELETE /api/cart/items/:id    → cart:3004/cart/items/:id (ลบสินค้า)
	//   DELETE /api/cart              → cart:3004/cart (ล้างตะกร้า)
	// Composition Route (specific first-match):
	//   GET    /api/cart              → BFF compose (cart + catalog variant images)
	app.Get("/api/cart", cartCompositionHandler.GetCart)
	app.All("/api/cart", cartHandler.Handle)
	app.All("/api/cart/*", cartHandler.Handle)

	// --- Order Service (REST Proxy) ---
	// What: forward order requests ไปยัง order-service (:3003)
	// Why:  order-service จัดการคำสั่งซื้อ รวมถึง payment และ admin cancel
	// ตัวอย่าง:
	//   POST /api/orders              → order:3003/orders (สร้าง order)
	//   GET  /api/orders/:id          → order:3003/orders/:id (ดู order)
	//   POST /api/orders/:id/cancel   → order:3003/orders/:id/cancel (ยกเลิก)
	//   POST /api/orders/:id/pay      → order:3003/orders/:id/pay (ชำระเงิน)
	app.All("/api/orders", orderHandler.Handle)
	app.All("/api/orders/*", orderHandler.Handle)

	// --- Catalog Service (REST Proxy) ---
	// What: forward catalog search requests ไปยัง catalog-service (:3005)
	// Why:  catalog-service เป็น read-optimized (MongoDB) สำหรับ search สินค้า
	//       แยกจาก product-service (PostgreSQL) ตาม CQRS pattern
	// ตัวอย่าง:
	//   GET /api/catalog/products          → catalog:3005/catalog/products (search)
	//   GET /api/catalog/products/:id      → catalog:3005/catalog/products/:id (detail)
	app.All("/api/catalog/*", catalogHandler.Handle)

	// --- Order History Service (REST Proxy) ---
	// What: forward order history requests ไปยัง order-history-service (:3006)
	// Why:  order-history เป็น read model (MongoDB) แยกจาก order-service ตาม CQRS
	//       เก็บ denormalized order data สำหรับ query เร็ว ไม่กระทบ write performance
	// ตัวอย่าง:
	//   GET /api/order-history             → order-history:3006/order-history (รายการ)
	//   GET /api/order-history/:orderId    → order-history:3006/order-history/:orderId
	app.All("/api/order-history", orderHistoryHandler.Handle)
	app.All("/api/order-history/*", orderHistoryHandler.Handle)

	// ═══════════════════════════════════════════════════════════════════════════
	// 7) Start Server
	// ═══════════════════════════════════════════════════════════════════════════
	port := getEnv("APP_PORT", "8080")
	log.Printf("BFF starting on port %s", port)
	log.Printf("  Auth:         %s", authURL)
	log.Printf("  Product:      %s", productURL)
	log.Printf("  Order:        %s", orderURL)
	log.Printf("  Cart:         %s", cartURL)
	log.Printf("  Catalog:      %s", catalogURL)
	log.Printf("  OrderHistory: %s", orderHistoryURL)
	log.Printf("  AI (gRPC):    %s", aiServiceURL)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("BFF failed to start: %v", err)
	}
}

// What: helper อ่าน env ด้วย fallback default value
// Why:  ลด boilerplate ไม่ต้องเขียน if-else ซ้ำทุกตัวแปร
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
