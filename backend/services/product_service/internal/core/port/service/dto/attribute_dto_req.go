package service

type CreateAttributeReq struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type UpdateAttributeReq struct {
	AttributeID uint   `json:"attribute_id" validate:"required"`
	Name        string `json:"name" validate:"required,min=1,max=100"`
}

type CreateAttributeValueReq struct {
	AttributeID uint   `json:"attribute_id" validate:"required"`
	Value       string `json:"value" validate:"required,min=1,max=100"`
}
