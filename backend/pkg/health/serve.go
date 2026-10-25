package health

import (
	"fmt"
	"logs"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ShutdownTimeout เวลาที่ให้ request ที่ค้างอยู่ทำงานให้เสร็จก่อนตัดทิ้ง
// WHY 10 วินาที? docker stop รอ 10s ก่อนส่ง SIGKILL → ต้องจบก่อนนั้น
//
//	ถ้าจะเพิ่มต้องตั้ง stop_grace_period ใน compose ให้มากกว่าค่านี้ด้วย
const ShutdownTimeout = 8 * time.Second

// Serve รัน app ที่ addr จนกว่าจะได้ SIGINT/SIGTERM แล้วปิดแบบ graceful:
//  1. /readyz ตอบ 503 → load balancer เลิกส่ง request ใหม่มา
//  2. หยุดรับ connection ใหม่ รอ request ที่ค้างอยู่ทำงานให้เสร็จ (สูงสุด ShutdownTimeout)
//  3. return → main ค่อยหยุด background worker (Kafka consumer, outbox) และปิด DB ผ่าน defer
//
// WHY ปิด HTTP ก่อน worker? request ที่กำลังทำอยู่อาจยังต้องเขียน DB / outbox
//
// คืน error ถ้า Listen พัง (เช่น port ถูกใช้อยู่) → main ต้อง exit แทนการค้างรอ signal ที่ไม่มีวันมา
func (c *Checker) Serve(app *fiber.App, addr string) error {
	listenErr := make(chan error, 1)
	go func() {
		listenErr <- app.Listen(addr)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-listenErr:
		return fmt.Errorf("%s: listen on %s: %w", c.service, addr, err)
	case sig := <-quit:
		logs.Info(fmt.Sprintf("%s: received %s, shutting down", c.service, sig))
	}

	c.MarkShuttingDown()
	if err := app.ShutdownWithTimeout(ShutdownTimeout); err != nil {
		return fmt.Errorf("%s: graceful shutdown: %w", c.service, err)
	}
	logs.Info(fmt.Sprintf("%s: HTTP server stopped", c.service))
	return nil
}
