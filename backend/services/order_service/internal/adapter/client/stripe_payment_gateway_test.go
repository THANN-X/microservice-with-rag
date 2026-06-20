// WHAT: Test suite สำหรับ StripeGateway
//
// ─── Credit Card Flow (ใหม่ — 3DS-compatible) ──────────────────────────────
// backend ไม่ confirm PaymentIntent เอง → คืน client_secret → frontend ทำ confirmCardPayment()
// ดังนั้น test ฝั่ง backend ตรวจได้แค่:
//   - PaymentIntent ถูกสร้างสำเร็จ (ChargeID มี prefix pi_)
//   - Status == "PENDING" (ยังรอ frontend confirm)
//   - ClientSecret ไม่ว่าง (frontend ต้องใช้ค่านี้)
//
// การทดสอบ decline / 3DS ทำได้ที่ frontend เท่านั้น ด้วย Stripe test cards:
//   4242 4242 4242 4242 → SUCCESS ทันที
//   4000 0027 6000 3184 → ต้องการ 3DS (PENDING → popup OTP)
//   4000 0000 0000 9995 → insufficient funds (FAILED)
//
// ─── Refund Test ─────────────────────────────────────────────────────────────
// ไม่สามารถ Refund PaymentIntent ที่ยังไม่ confirmed ได้
// test นี้จึงต้อง confirm PI โดยตรงผ่าน Stripe API (ใช้ test pm_card_visa)
// ก่อนแล้วค่อย test Refund — แยก concern ชัดเจน
//
// การรัน test:
//   STRIPE_TEST_KEY=sk_test_xxxx go test ./internal/adapter/client/... -v -run TestStripe
//
// ถ้าไม่มี STRIPE_TEST_KEY → test จะ skip อัตโนมัติ (ไม่ fail CI ถ้ายังไม่มี key)
package client_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"

	"order_service/internal/adapter/client"
	gateway "order_service/internal/core/port/gateway"
)

// getTestGateway สร้าง StripeGateway จาก STRIPE_TEST_KEY env var
// skip test ถ้าไม่มี key (เหมาะกับ CI ที่ยังไม่ได้ configure)
func getTestGateway(t *testing.T) gateway.PaymentGateway {
	t.Helper()
	key := os.Getenv("STRIPE_TEST_KEY")
	if key == "" {
		t.Skip("STRIPE_TEST_KEY not set — skipping Stripe integration test")
	}
	// webhookSecret ใช้ค่าว่างใน unit test (ไม่มีการทดสอบ webhook ใน test นี้)
	return client.NewStripeGateway(key, "")
}

// initTestStripeKey set stripe.Key สำหรับ helper ที่เรียก Stripe API โดยตรง
func initTestStripeKey(t *testing.T) {
	t.Helper()
	key := os.Getenv("STRIPE_TEST_KEY")
	if key == "" {
		t.Skip("STRIPE_TEST_KEY not set — skipping Stripe integration test")
	}
	stripe.Key = key
}

// TestStripeCharge_CreditCard_ReturnsPendingWithSecret ตรวจว่า Charge() สร้าง PaymentIntent
// และคืน PENDING + ClientSecret (frontend จะเป็นผู้ confirm + จัดการ 3DS เอง)
func TestStripeCharge_CreditCard_ReturnsPendingWithSecret(t *testing.T) {
	gw := getTestGateway(t)

	req := &gateway.ChargeRequest{
		OrderID:    "order-test-001",
		CustomerID: 1,
		Amount:     100.00, // 100 บาท
		Currency:   "THB",
		Token:      "",     // ไม่ต้องการ token — frontend จะ confirm ด้วย CardElement
		Method:     "CREDIT_CARD",
	}

	res, err := gw.Charge(context.Background(), req)
	if err != nil {
		t.Fatalf("Charge() error = %v", err)
	}

	if !strings.HasPrefix(res.ChargeID, "pi_") {
		t.Errorf("ChargeID = %q, want prefix pi_", res.ChargeID)
	}
	if res.Status != "PENDING" {
		t.Errorf("Status = %q, want PENDING (frontend ยังไม่ confirm)", res.Status)
	}
	if res.ClientSecret == "" {
		t.Error("ClientSecret must not be empty — frontend ต้องการ secret นี้เพื่อ confirmCardPayment()")
	}

	t.Logf("ChargeID: %s | Status: %s | ClientSecret prefix: %s...", res.ChargeID, res.Status, res.ClientSecret[:20])
}

// TestStripeCharge_PromptPay_ReturnsPendingWithSecret ตรวจ PromptPay flow (ไม่เปลี่ยน)
func TestStripeCharge_PromptPay_ReturnsPendingWithSecret(t *testing.T) {
	gw := getTestGateway(t)

	req := &gateway.ChargeRequest{
		OrderID:    "order-test-promptpay-001",
		CustomerID: 1,
		Amount:     100.00,
		Currency:   "THB",
		Token:      "",
		Method:     "PROMPTPAY",
	}

	res, err := gw.Charge(context.Background(), req)
	if err != nil {
		t.Fatalf("Charge() error = %v", err)
	}

	if !strings.HasPrefix(res.ChargeID, "pi_") {
		t.Errorf("ChargeID = %q, want prefix pi_", res.ChargeID)
	}
	if res.Status != "PENDING" {
		t.Errorf("Status = %q, want PENDING", res.Status)
	}
	if res.ClientSecret == "" {
		t.Error("ClientSecret must not be empty")
	}

	t.Logf("PromptPay ChargeID: %s | ClientSecret prefix: %s...", res.ChargeID, res.ClientSecret[:20])
}

// TestStripeRefund_Full ทดสอบ full refund
//
// WHY ต้อง confirm PI โดยตรง?
//   - Refund ทำได้เฉพาะ PaymentIntent ที่ status == "succeeded"
//   - Charge() ใหม่คืน PENDING (รอ frontend confirm) → refund ทันทีไม่ได้
//   - ใน test เราจึง confirm PI ด้วย Stripe API โดยตรง (ใช้ test PM pm_card_visa)
func TestStripeRefund_Full(t *testing.T) {
	initTestStripeKey(t)
	gw := getTestGateway(t)

	// Step 1: สร้าง PaymentIntent ผ่าน gateway
	chargeRes, err := gw.Charge(context.Background(), &gateway.ChargeRequest{
		OrderID:    "order-test-refund-001",
		CustomerID: 1,
		Amount:     250.00,
		Currency:   "THB",
		Method:     "CREDIT_CARD",
	})
	if err != nil {
		t.Fatalf("Charge() error = %v", err)
	}

	// Step 2: Confirm PaymentIntent โดยตรงผ่าน Stripe API (simulate frontend confirmCardPayment)
	// WHY pm_card_visa? — test PaymentMethod ID ที่ Stripe กำหนดไว้สำหรับ test mode
	confirmedPI, err := paymentintent.Confirm(chargeRes.ChargeID, &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String("pm_card_visa"),
		ReturnURL:     stripe.String("https://example.com/return"),
	})
	if err != nil {
		t.Fatalf("paymentintent.Confirm() error = %v", err)
	}
	if confirmedPI.Status != stripe.PaymentIntentStatusSucceeded {
		t.Fatalf("PI status = %s, want succeeded", confirmedPI.Status)
	}
	t.Logf("PI confirmed: %s (status: %s)", confirmedPI.ID, confirmedPI.Status)

	// Step 3: Full refund ผ่าน gateway
	err = gw.Refund(context.Background(), chargeRes.ChargeID, 0)
	if err != nil {
		t.Fatalf("Refund() error = %v", err)
	}
	t.Logf("Full refund successful for %s", chargeRes.ChargeID)
}

// TestStripeRefund_Partial ทดสอบ partial refund
func TestStripeRefund_Partial(t *testing.T) {
	initTestStripeKey(t)
	gw := getTestGateway(t)

	chargeRes, err := gw.Charge(context.Background(), &gateway.ChargeRequest{
		OrderID:    "order-test-partial-refund",
		CustomerID: 1,
		Amount:     500.00,
		Currency:   "THB",
		Method:     "CREDIT_CARD",
	})
	if err != nil {
		t.Fatalf("Charge() error = %v", err)
	}

	_, err = paymentintent.Confirm(chargeRes.ChargeID, &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String("pm_card_visa"),
		ReturnURL:     stripe.String("https://example.com/return"),
	})
	if err != nil {
		t.Fatalf("paymentintent.Confirm() error = %v", err)
	}

	// Partial refund 200 บาท
	err = gw.Refund(context.Background(), chargeRes.ChargeID, 200.00)
	if err != nil {
		t.Fatalf("Partial Refund() error = %v", err)
	}
	t.Logf("Partial refund (200 THB) successful for %s", chargeRes.ChargeID)
}

// TestStripeCharge_MetadataContainsOrderID ตรวจว่า metadata order_id ถูก set ถูกต้อง
// (สำคัญ: webhook ใช้ metadata["order_id"] เพื่อ link กลับ order)
func TestStripeCharge_MetadataContainsOrderID(t *testing.T) {
	initTestStripeKey(t)

	orderID := "order-meta-check-001"
	stripe.Key = os.Getenv("STRIPE_TEST_KEY")

	// สร้าง PI โดยตรงเพื่อตรวจ metadata
	gw := getTestGateway(t)
	res, err := gw.Charge(context.Background(), &gateway.ChargeRequest{
		OrderID:    orderID,
		CustomerID: 42,
		Amount:     99.00,
		Currency:   "THB",
		Method:     "CREDIT_CARD",
	})
	if err != nil {
		t.Fatalf("Charge() error = %v", err)
	}

	// ดึง PI จาก Stripe เพื่อตรวจ metadata
	pi, err := paymentintent.Get(res.ChargeID, nil)
	if err != nil {
		t.Fatalf("paymentintent.Get() error = %v", err)
	}

	if pi.Metadata["order_id"] != orderID {
		t.Errorf("metadata[order_id] = %q, want %q", pi.Metadata["order_id"], orderID)
	}
	if pi.Metadata["customer_id"] != fmt.Sprintf("%d", 42) {
		t.Errorf("metadata[customer_id] = %q, want %q", pi.Metadata["customer_id"], "42")
	}
	t.Logf("Metadata verified: order_id=%s, customer_id=%s", pi.Metadata["order_id"], pi.Metadata["customer_id"])
}
