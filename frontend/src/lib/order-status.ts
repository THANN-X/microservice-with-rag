import type { OrderStatus } from "./types";

// ─── Order status badge ───
// แหล่งความจริงเดียวของ label/สีสถานะคำสั่งซื้อ ใช้ร่วมกันทุกหน้า
//
// WHY เป็น Record<OrderStatus, …> ไม่ใช่ Record<string, …>?
//   - เดิมแต่ละหน้ามี map ของตัวเอง (copy-paste 6 ชุด) แล้ว drift กัน
//     บางชุดขาด AWAITING_PAYMENT/PAID → ตกไปที่ fallback แล้วโชว์ "รอดำเนินการ"
//     ทั้งที่ลูกค้าจ่ายเงินแล้ว
//   - พิมพ์เป็น Record<OrderStatus, …> ทำให้ compiler ฟ้องทันทีถ้าเพิ่มสถานะใหม่
//     ใน union แล้วลืมใส่ badge (และฟ้องด้วยถ้าใส่ key ที่ backend ไม่มี เช่น SHIPPED)
//
// NOTE: ชุดสถานะต้องตรงกับ order_service/internal/core/domain/order.go

/** Badge สำหรับ Admin console — palette แบบ flat ตามดีไซน์ฝั่ง admin */
export const ORDER_STATUS_ADMIN: Record<OrderStatus, { label: string; cls: string }> = {
  PENDING: { label: "รอดำเนินการ", cls: "bg-amber-50 text-amber-700" },
  CONFIRMED: { label: "ยืนยันแล้ว", cls: "bg-blue-50 text-blue-700" },
  AWAITING_PAYMENT: { label: "รอชำระเงิน", cls: "bg-orange-50 text-orange-700" },
  PAID: { label: "ชำระเงินแล้ว", cls: "bg-emerald-50 text-emerald-700" },
  COMPLETED: { label: "สำเร็จ", cls: "bg-teal-50 text-teal-700" },
  CANCELLED: { label: "ยกเลิก", cls: "bg-red-50 text-red-700" },
};

/** Badge สำหรับหน้าร้าน — ใช้ theme token ของ design system */
export const ORDER_STATUS_SHOP: Record<OrderStatus, { label: string; color: string }> = {
  PENDING: { label: "รอดำเนินการ", color: "bg-amber-100 text-amber-800" },
  CONFIRMED: { label: "ยืนยันแล้ว", color: "bg-blue-100 text-blue-800" },
  AWAITING_PAYMENT: { label: "รอชำระเงิน", color: "bg-amber-100 text-amber-800" },
  PAID: { label: "ชำระเงินแล้ว", color: "bg-emerald-100 text-emerald-800" },
  COMPLETED: { label: "จัดส่งสำเร็จ", color: "bg-tertiary-container/30 text-on-tertiary-container" },
  CANCELLED: { label: "ยกเลิก", color: "bg-error-container/30 text-on-error-container" },
};

// WHY fallback ไม่ใช่ PENDING?
//   - status มาจาก API ตอน runtime → เป็นค่านอก union ได้ถ้า backend เพิ่มสถานะใหม่
//   - การ fallback ไปที่ PENDING = โชว์ข้อมูลผิดแบบเงียบ ๆ (บั๊กเดิม)
//   - โชว์ค่าดิบแทน ทำให้เห็นทันทีว่ามีสถานะที่ frontend ยังไม่รู้จัก

export function adminOrderStatus(status: string): { label: string; cls: string } {
  return (
    ORDER_STATUS_ADMIN[status as OrderStatus] ?? {
      label: status || "ไม่ทราบสถานะ",
      cls: "bg-surface-highest text-secondary",
    }
  );
}

export function shopOrderStatus(status: string): { label: string; color: string } {
  return (
    ORDER_STATUS_SHOP[status as OrderStatus] ?? {
      label: status || "ไม่ทราบสถานะ",
      color: "bg-surface-highest text-secondary",
    }
  );
}
