package dto

type OrderHistoryRes struct {
	OrderID         string                  `json:"order_id"`
	CustomerID      uint                    `json:"customer_id"`
	Status          string                  `json:"status"`
	TotalAmount     float64                 `json:"total_amount"`
	Items           []OrderHistoryItemRes   `json:"items"`
	ShippingAddress ShippingAddressRes      `json:"shipping_address"`
	Note            string                  `json:"note"`
	CancelReason    string                  `json:"cancel_reason,omitempty"`
	CreatedAt       string                  `json:"created_at"`
	UpdatedAt       string                  `json:"updated_at"`
}

type OrderHistoryItemRes struct {
	VariantID uint    `json:"variant_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
}

type ShippingAddressRes struct {
	FullName    string `json:"full_name"`
	Phone       string `json:"phone"`
	AddressLine string `json:"address_line"`
	SubDistrict string `json:"sub_district"`
	District    string `json:"district"`
	Province    string `json:"province"`
	PostalCode  string `json:"postal_code"`
}

type ListOrderHistoryReq struct {
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Status string `query:"status"`
}

type OrderHistoryListRes struct {
	Items      []OrderHistoryRes `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}
