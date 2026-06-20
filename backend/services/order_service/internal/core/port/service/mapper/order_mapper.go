// WHAT: Mapper functions แปลง DTOs ↔ Domain Objects
// WHY แยก mapper ออกมา?
//   - ทำให้ service code สะอาด ไม่มี mapping boilerplate
//   - Testable ได้ง่าย (pure functions)
//   - ถ้า DTO หรือ domain เปลี่ยน แก้ที่นี่ที่เดียว
package mapper

import (
	"order_service/internal/core/domain"
	dto "order_service/internal/core/port/service/dto"
)

// ToOrderItemsDomain แปลง []CreateOrderItemReq → []domain.OrderItem
// WHY ไม่มี UnitPrice / ProductName / ImageURL ใน partial items นี้?
//   - Client ส่งมาแค่ VariantID + Quantity
//   - PlaceOrder service เติม UnitPrice, ProductName, VariantName, ImageURL
//     จาก catalog_service ก่อนส่ง items เข้า domain.NewOrder()
func ToOrderItemsDomain(reqs []dto.CreateOrderItemReq) []domain.OrderItem {
	items := make([]domain.OrderItem, len(reqs))
	for i, r := range reqs {
		items[i] = domain.OrderItem{
			VariantID: r.VariantID,
			Quantity:  r.Quantity,
		}
	}
	return items
}

// ToShippingAddressDomain แปลง ShippingAddressReq → domain.ShippingAddress
func ToShippingAddressDomain(req dto.ShippingAddressReq) domain.ShippingAddress {
	return domain.ShippingAddress{
		FullName:    req.FullName,
		Phone:       req.Phone,
		AddressLine: req.AddressLine,
		SubDistrict: req.SubDistrict,
		District:    req.District,
		Province:    req.Province,
		PostalCode:  req.PostalCode,
	}
}

// ToOrderRes แปลง *domain.Order → *dto.OrderRes
func ToOrderRes(o *domain.Order) *dto.OrderRes {
	if o == nil {
		return nil
	}

	items := make([]dto.OrderItemRes, len(o.Items))
	for i, item := range o.Items {
		items[i] = dto.OrderItemRes{
			ID:          item.ID,
			VariantID:   item.VariantID,
			ProductName: item.ProductName,
			VariantName: item.VariantName,
			ImageURL:    item.ImageURL,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Subtotal:    item.UnitPrice * float64(item.Quantity),
		}
	}

	return &dto.OrderRes{
		ID:          o.ID,
		CustomerID:  o.CustomerID,
		Status:      string(o.Status),
		TotalAmount: o.TotalAmount,
		Items:       items,
		ShippingAddress: dto.ShippingAddressRes{
			FullName:    o.ShippingAddress.FullName,
			Phone:       o.ShippingAddress.Phone,
			AddressLine: o.ShippingAddress.AddressLine,
			SubDistrict: o.ShippingAddress.SubDistrict,
			District:    o.ShippingAddress.District,
			Province:    o.ShippingAddress.Province,
			PostalCode:  o.ShippingAddress.PostalCode,
		},
		Note:      o.Note,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

func ToPaymentRes(p *domain.Payment) *dto.PaymentRes {
	if p == nil {
		return nil
	}
	return &dto.PaymentRes{
		ID:              p.ID,
		OrderID:         p.OrderID,
		Amount:          p.Amount,
		Currency:        p.Currency,
		Status:          string(p.Status),
		Gateway:         p.Gateway,
		GatewayChargeID: p.GatewayChargeID,
		PaymentMethod:   p.PaymentMethod,
		PaidAt:          p.PaidAt,
		CreatedAt:       p.CreatedAt,
	}
}
