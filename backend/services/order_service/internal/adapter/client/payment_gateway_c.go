// WHAT: Stub Payment Gateway สำหรับ development/testing
// TODO: แทนที่ด้วย Stripe (github.com/stripe/stripe-go)
//       หรือ Omise (github.com/omise/omise-go) สำหรับ production
package client

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	gateway "order_service/internal/core/port/gateway"
)

type StubPaymentGateway struct{}

func NewStubPaymentGateway() gateway.PaymentGateway {
	return &StubPaymentGateway{}
}

// Charge จำลองการ charge เงิน — always returns SUCCESS
func (g *StubPaymentGateway) Charge(_ context.Context, _ *gateway.ChargeRequest) (*gateway.ChargeResponse, error) {
	return &gateway.ChargeResponse{
		ChargeID: "ch_stub_" + uuid.NewString()[:8],
		Status:   "SUCCESS",
	}, nil
}

// VerifyWebhook จำลองการ verify webhook — parse JSON body ตรงๆ
// TODO: production ต้อง verify signature ด้วย HMAC/RSA จาก gateway
func (g *StubPaymentGateway) VerifyWebhook(_ string, payload []byte) (*gateway.WebhookEvent, error) {
	var evt gateway.WebhookEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, err
	}
	return &evt, nil
}

// Refund จำลองการ refund — always succeeds
func (g *StubPaymentGateway) Refund(_ context.Context, _ string, _ float64) error {
	return nil
}
