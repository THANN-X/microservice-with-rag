package dto

type ChatRequest struct {
	Message   string `json:"message" validate:"required,min=1,max=1000"`
	SessionID string `json:"session_id"`
}

type ChatResponseChunk struct {
	EventType   string   `json:"event_type"`
	TextContent string   `json:"text_content,omitempty"`
	ProductIDs  []uint32 `json:"product_ids,omitempty"`
}
