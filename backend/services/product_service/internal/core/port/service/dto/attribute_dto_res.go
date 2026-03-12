package service

type AttributeValueRes struct {
	ID    uint   `json:"id"`
	Value string `json:"value"`
}

type AttributeRes struct {
	ID     uint                `json:"id"`
	Name   string              `json:"name"`
	Values []AttributeValueRes `json:"values"`
}
