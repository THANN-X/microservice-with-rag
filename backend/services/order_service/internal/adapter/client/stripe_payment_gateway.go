// WHAT: StripeGateway — implementation ของ PaymentGateway interface โดยใช้ Stripe
//
// WHY Stripe?
//   - รองรับ 135+ สกุลเงิน รวม THB
//   - Test mode ครบ: ใช้ sk_test_... แทน sk_live_... ไม่มีเงินจริงถูกหัก
//   - Stripe.js สร้าง PaymentMethod token จาก frontend → ส่ง token_id มาที่ backend
//     (backend ไม่รับเลขบัตรโดยตรง ลด PCI-DSS scope)
//
// Flow (Credit Card):
//
//	Backend → Charge(method="CREDIT_CARD") → PaymentIntent(pending) → client_secret
//	→ Frontend stripe.confirmCardPayment(client_secret, {card}) → Stripe จัดการ 3DS/OTP เอง
//	→ Stripe webhook payment_intent.succeeded → backend update
//
// Flow (PromptPay):
//
//	Backend → Charge(method="PROMPTPAY") → PaymentIntent(pending)
//	→ Stripe ส่ง QR back → customer scan → Stripe webhook → VerifyWebhook
//
// Test cards (ใช้ใน test mode):
//
//	pm_card_visa                  → SUCCESS ทันที
//	pm_card_visa_debit            → SUCCESS ทันที
//	pm_card_authenticationRequired → ต้องการ 3DS
//	pm_card_chargeDeclined        → FAILED (declined)
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/paymentmethod"
	striprefund "github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/webhook"

	gateway "order_service/internal/core/port/gateway"
)

// StripeGateway implements gateway.PaymentGateway โดยใช้ Stripe PaymentIntents API
type StripeGateway struct {
	secretKey     string
	webhookSecret string
}

// NewStripeGateway สร้าง StripeGateway และ set global stripe.Key
//
// secretKey     — ได้จาก Stripe Dashboard → Developers → API keys
//
//	Test mode: sk_test_xxxx   (ไม่มีเงินจริงถูกหัก)
//	Live mode: sk_live_xxxx   (production เท่านั้น)
//
// webhookSecret — ได้จาก Stripe Dashboard → Developers → Webhooks → Signing secret
//
//	ใช้ verify ว่า webhook request มาจาก Stripe จริง (HMAC-SHA256)
func NewStripeGateway(secretKey, webhookSecret string) gateway.PaymentGateway {
	stripe.Key = secretKey
	return &StripeGateway{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
	}
}

// Charge สร้าง Stripe PaymentIntent
//
// Credit Card (req.Method != "PROMPTPAY"):
//
//	สร้าง PaymentIntent (ไม่ confirm, ไม่ต้องการ token) → return PENDING + ClientSecret
//	Frontend ใช้ stripe.confirmCardPayment(clientSecret, {card}) → Stripe จัดการ 3DS/OTP เอง
//
// PromptPay (req.Method == "PROMPTPAY"):
//
//	ไม่ต้องการ token จาก frontend
//	สร้าง PaymentIntent with payment_method_types:["promptpay"], ไม่ confirm
//	return PENDING + ClientSecret → frontend ใช้ stripe.confirmPromptPayPayment(clientSecret) → QR
func (g *StripeGateway) Charge(ctx context.Context, req *gateway.ChargeRequest) (*gateway.ChargeResponse, error) {
	amountSatang := int64(req.Amount * 100)

	if req.Method == "PROMPTPAY" {
		return g.chargePromptPay(amountSatang, req)
	}
	return g.chargeCard(amountSatang, req)
}

// chargeCard สร้าง PaymentIntent สำหรับบัตรเครดิต/เดบิต (ไม่ confirm — รอ frontend)
//
// WHY ไม่ Confirm ทันที?
//   - บัตรส่วนใหญ่ต้องการ 3D Secure (SCA) → Stripe.js ต้องจัดการ popup OTP ฝั่ง client
//   - ถ้า Confirm ฝั่ง server โดยไม่มี return_url → บัตร 3DS จะ fail ทันที
//   - Flow ที่ถูกต้อง: backend คืน client_secret → frontend confirmCardPayment() → 3DS handled
func (g *StripeGateway) chargeCard(amountSatang int64, req *gateway.ChargeRequest) (*gateway.ChargeResponse, error) {
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(amountSatang),
		Currency:           stripe.String(strings.ToLower(req.Currency)),
		PaymentMethodTypes: []*string{stripe.String("card")},
		Metadata: map[string]string{
			"order_id":    req.OrderID,
			"customer_id": fmt.Sprintf("%d", req.CustomerID),
			"method":      req.Method,
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			return nil, fmt.Errorf("stripe charge failed [%s]: %s", stripeErr.Code, stripeErr.Msg)
		}
		return nil, fmt.Errorf("stripe charge failed: %w", err)
	}

	// PaymentIntent เพิ่งสร้าง → status = "requires_payment_method"
	// Frontend จะ confirmCardPayment() → status เปลี่ยนเป็น "requires_action" (3DS) หรือ "succeeded"
	return &gateway.ChargeResponse{
		ChargeID:     pi.ID,
		Status:       "PENDING",
		ClientSecret: pi.ClientSecret,
	}, nil
}

// chargePromptPay สร้าง PaymentIntent สำหรับ PromptPay + confirm ทันที → ได้ QR URL
//
// WHY confirm ที่ backend แทน frontend?
//   - PromptPay ไม่ต้องการ 3DS / user-input — สร้าง PM ได้โดยตรงที่ server
//   - หลีกเลี่ยง error "payment method of type promptpay was expected" จาก frontend
//   - frontend แค่รับ QR URL ไปแสดง ไม่ต้องรู้จัก Stripe.js PromptPay API
//
// Flow:
//  1. สร้าง PromptPay PaymentMethod
//  2. สร้าง + confirm PaymentIntent พร้อม PM → status = "requires_action"
//  3. ดึง QR URL จาก next_action.promptpay_display_qr_code.image_url_png
func (g *StripeGateway) chargePromptPay(amountSatang int64, req *gateway.ChargeRequest) (*gateway.ChargeResponse, error) {
	// 1. สร้าง PromptPay PaymentMethod
	// WHY ต้องใส่ BillingDetails.Email? — Stripe กำหนด billing_details[email] เป็น required
	//   ใช้ placeholder email เพราะ PromptPay ไม่ส่ง email จริงๆ ให้ลูกค้า
	pm, err := paymentmethod.New(&stripe.PaymentMethodParams{
		Type: stripe.String("promptpay"),
		BillingDetails: &stripe.PaymentMethodBillingDetailsParams{
			Email: stripe.String(fmt.Sprintf("customer_%d@checkout.local", req.CustomerID)),
		},
	})
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			return nil, fmt.Errorf("stripe create promptpay pm failed [%s]: %s", stripeErr.Code, stripeErr.Msg)
		}
		return nil, fmt.Errorf("stripe create promptpay pm failed: %w", err)
	}

	// 2. สร้าง + confirm PaymentIntent
	// WHY ต้องมี ReturnURL? — Stripe กำหนดให้ redirect-based payment ต้องมี return_url
	//   ใช้ placeholder เพราะ QR flow ไม่ได้ redirect browser จริงๆ (ลูกค้า scan ด้วยมือถือ)
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(amountSatang),
		Currency:           stripe.String(strings.ToLower(req.Currency)),
		PaymentMethod:      stripe.String(pm.ID),
		PaymentMethodTypes: []*string{stripe.String("promptpay")},
		Confirm:            stripe.Bool(true),
		ReturnURL:          stripe.String("https://placeholder.example.com/payment-complete"),
		Metadata: map[string]string{
			"order_id":    req.OrderID,
			"customer_id": fmt.Sprintf("%d", req.CustomerID),
			"method":      "PROMPTPAY",
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			return nil, fmt.Errorf("stripe promptpay failed [%s]: %s", stripeErr.Code, stripeErr.Msg)
		}
		return nil, fmt.Errorf("stripe promptpay failed: %w", err)
	}

	// 3. ดึง QR URL จาก next_action
	var qrURL string
	if pi.NextAction != nil && pi.NextAction.PromptPayDisplayQRCode != nil {
		qrURL = pi.NextAction.PromptPayDisplayQRCode.ImageURLPNG
	}

	return &gateway.ChargeResponse{
		ChargeID:     pi.ID,
		Status:       "PENDING",
		ClientSecret: pi.ClientSecret,
		QRImageURL:   qrURL,
	}, nil
}

// VerifyWebhook ตรวจสอบ Stripe-Signature header และ parse event
//
// WHY ต้อง verify signature ก่อน parse?
//   - ป้องกัน attacker ส่ง fake webhook มาบอกว่า "order XXX จ่ายแล้ว"
//   - Stripe sign payload ด้วย HMAC-SHA256 + webhookSecret
//   - webhook.ConstructEvent verify และ return parsed Event
//
// signature — ค่าจาก HTTP Header "Stripe-Signature"
// payload   — raw request body (ต้องเป็น raw bytes ก่อน JSON parse)
func (g *StripeGateway) VerifyWebhook(signature string, payload []byte) (*gateway.WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, signature, g.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, fmt.Errorf("failed to parse payment_intent: %w", err)
		}
		return &gateway.WebhookEvent{
			ChargeID: pi.ID,
			OrderID:  pi.Metadata["order_id"],
			Status:   "SUCCESS",
		}, nil

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, fmt.Errorf("failed to parse payment_intent: %w", err)
		}
		return &gateway.WebhookEvent{
			ChargeID: pi.ID,
			OrderID:  pi.Metadata["order_id"],
			Status:   "FAILED",
		}, nil

	default:
		// event types อื่นๆ ที่ไม่ได้ handle — return nil error เพื่อ return 200 ให้ Stripe
		// (Stripe retry ถ้าได้รับ non-2xx)
		return nil, fmt.Errorf("unhandled stripe event type: %s", event.Type)
	}
}

// Refund คืนเงิน full หรือ partial amount ผ่าน Stripe Refunds API
//
// chargeID — PaymentIntent ID (pi_xxxx) ที่ได้จาก Charge()
// amount   — จำนวนที่ refund (บาท); ถ้า 0 → full refund
func (g *StripeGateway) Refund(ctx context.Context, chargeID string, amount float64) error {
	params := &stripe.RefundParams{
		// PaymentIntent แทน Charge เพราะเราสร้าง PaymentIntent (ไม่ใช่ legacy Charge object)
		PaymentIntent: stripe.String(chargeID),
	}
	if amount > 0 {
		params.Amount = stripe.Int64(int64(amount * 100))
	}

	_, err := striprefund.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			return fmt.Errorf("stripe refund failed [%s]: %s", stripeErr.Code, stripeErr.Msg)
		}
		return fmt.Errorf("stripe refund failed: %w", err)
	}
	return nil
}
