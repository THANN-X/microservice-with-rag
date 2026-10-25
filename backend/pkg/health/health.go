// WHAT: package health ให้ทุก service มี /healthz (liveness) และ /readyz (readiness) แบบเดียวกัน
// WHY แยก 2 endpoint?
//   - /healthz ตอบว่า "process ยังไม่ค้าง" → ถ้าพัง orchestrator ควร restart container
//   - /readyz ตอบว่า "พร้อมรับ traffic ไหม" (DB ต่อได้, ไม่ได้กำลังปิดตัว) → ถ้าพังแค่หยุดส่ง traffic มา ไม่ต้อง restart
//   - ถ้ารวมเป็นอันเดียว DB ล่มทีเดียว → ทุก service โดน restart วนพร้อมกัน ทั้งที่ restart ไม่ได้ช่วยอะไร
package health

import (
	"context"
	"database/sql"
	"fmt"
	"logs"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
)

const checkTimeout = 2 * time.Second

// Check คืน nil เมื่อ dependency พร้อมใช้งาน
type Check func(ctx context.Context) error

// Checker เก็บรายการ dependency ที่ /readyz ต้องตรวจ
type Checker struct {
	service      string
	names        []string
	checks       map[string]Check
	shuttingDown atomic.Bool
}

func New(service string) *Checker {
	return &Checker{service: service, checks: map[string]Check{}}
}

// Add เพิ่ม dependency ที่ต้องพร้อมก่อนรับ traffic
func (c *Checker) Add(name string, check Check) *Checker {
	c.names = append(c.names, name)
	c.checks[name] = check
	return c
}

// MarkShuttingDown ให้ /readyz ตอบ 503 ทันทีตอนเริ่มปิด service
// WHY: load balancer จะเลิกส่ง request ใหม่มา ระหว่างที่ request เดิมกำลังทำให้เสร็จ
func (c *Checker) MarkShuttingDown() {
	c.shuttingDown.Store(true)
}

// Register ผูก /healthz และ /readyz เข้ากับ app
// IMPORTANT: เรียกก่อน app.Use(...) ของ auth / rate limiter
//
//	ไม่งั้น healthcheck ของ docker โดน 401 หรือโดนนับเป็น request ใน rate limit
func (c *Checker) Register(app *fiber.App) {
	app.Get("/healthz", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/readyz", c.handleReady)
}

func (c *Checker) handleReady(ctx *fiber.Ctx) error {
	if c.shuttingDown.Load() {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "shutting_down"})
	}

	results, ok := c.run(ctx.UserContext())
	status, code := "ok", fiber.StatusOK
	if !ok {
		status, code = "unavailable", fiber.StatusServiceUnavailable
	}
	return ctx.Status(code).JSON(fiber.Map{"status": status, "checks": results})
}

// run ตรวจทุก dependency พร้อมกัน แต่ละตัวมี timeout ของตัวเอง
// WHY ไม่ส่ง error message กลับไปใน response?
//   - BFF เปิดสู่ public → error ของ driver อาจมี hostname / user ของ DB ติดไป → log ฝั่ง server แทน
func (c *Checker) run(parent context.Context) (map[string]string, bool) {
	results := make(map[string]string, len(c.names))
	var mu sync.Mutex
	var wg sync.WaitGroup
	ok := true

	for _, name := range c.names {
		wg.Add(1)
		go func(name string, check Check) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(parent, checkTimeout)
			defer cancel()

			err := check(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				ok = false
				results[name] = "unavailable"
				logs.Warn(fmt.Sprintf("%s: readiness check %q failed: %v", c.service, name, err))
				return
			}
			results[name] = "ok"
		}(name, c.checks[name])
	}
	wg.Wait()
	return results, ok
}

// SQL สร้าง Check จาก *sql.DB (GORM ใช้ db.DB() เพื่อดึงออกมา)
func SQL(db *sql.DB) Check {
	return func(ctx context.Context) error {
		return db.PingContext(ctx)
	}
}
