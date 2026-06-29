// WHAT: ChatService — BFF thin service layer สำหรับ AI chat feature
//
// WHY ต้องมี service layer ทั้งที่แค่ pass-through ไป AIClientPort?
//   - BFF architecture: handler → service → port (hexagonal)
//   - ถ้าต้องการเพิ่ม logic เช่น rate limit, session injection, logging หรือ fallback
//     ทำได้ที่ service layer โดยไม่แตะ handler หรือ gRPC client
//   - AIClientPort เป็น interface → swap ระหว่าง gRPC client (Python ai_service) กับ mock ได้ตอน test
package service

import (
	"bff/internal/core/dto"
	"bff/internal/core/port"
	"context"
)

type ChatService struct {
	aiClient port.AIClientPort
}

func NewChatService(aiClient port.AIClientPort) *ChatService {
	return &ChatService{aiClient: aiClient}
}

func (s *ChatService) ChatStream(ctx context.Context, req *dto.ChatRequest, ch chan<- *dto.ChatResponseChunk) error {
	return s.aiClient.ChatStream(ctx, req.Message, req.SessionID, ch)
}
