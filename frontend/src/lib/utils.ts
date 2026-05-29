/**
 * utils.ts — Utility functions ทั่วไปที่ใช้ข้ามทั้งแอป
 *
 * What: รวม helper functions ที่ไม่เกี่ยวกับ business logic เฉพาะส่วน
 * Why:  ป้องกันการเขียนโค้ดซ้ำ และทำให้ทดสอบได้ง่าย
 * How:  import ตรงมาใช้ได้เลย ไม่มี side effect
 */
import { clsx, type ClassValue } from "clsx";

/**
 * What: รวม Tailwind class names โดยกรอง falsy ออก
 * Why:  ใช้สร้าง className แบบ conditional ได้สะดวก เช่น cn("base", isActive && "active")
 * How:  ส่ง class ผ่าน clsx() ซึ่งจัดการ string/array/object ได้ทุกรูปแบบ
 */
export function cn(...inputs: ClassValue[]) {
  return clsx(inputs);
}

/** Format a number into Thai Baht display: ฿ 1,250
 * What: แปลงตัวเลขเป็น string ราคาบาทไทยพร้อม locale formatting
 * Why:  ใช้แสดงราคาสินค้า/ยอดสั่งซื้อทั่วทั้งแอปให้รูปแบบเดียวกัน
 * How:  ใช้ Intl.NumberFormat ผ่าน toLocaleString("th-TH") เพิ่มเครื่องหมาย ฿ ข้างหน้า
 */
export function formatBaht(amount: number): string {
  return `฿ ${amount.toLocaleString("th-TH", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  })}`;
}

/** Get lowest price from a variant list
 * What: หาราคาต่ำสุดจากทุก variant ของสินค้า
 * Why:  product card แสดงแค่ "ราคาเริ่มต้น" เพื่อไม่ให้ UI รก
 * How:  spread variants ลงใน Math.min() — คืน 0 ถ้าไม่มี variant เลย
 */
export function getMinPrice(variants: { price: number }[]): number {
  if (!variants.length) return 0;
  return Math.min(...variants.map((v) => v.price));
}

/** Truncate text
 * What: ตัดข้อความให้สั้นลงและใส่ "…" ถ้ายาวเกิน max
 * Why:  ป้องกัน text ล้น card/container ในหน้า product listing
 * How:  ตรวจความยาวก่อน — ถ้าไม่เกิน max ส่งกลับตรง ๆ ไม่ต้องทำอะไรเพิ่ม
 */
export function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  return text.slice(0, max) + "…";
}
