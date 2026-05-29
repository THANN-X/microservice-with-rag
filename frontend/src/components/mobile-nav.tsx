"use client";

/**
 * MobileNav — bottom navigation bar (mobile only, md ลงมา)
 *
 * What: Tab bar ด้านล่างทำหน้าที่เดียวกับ Sidebar บน desktop
 * Why:  mobile UX standard — navigation ด้านล่างเข้าถึงได้ง่ายด้วยนิ้วหัวแม่มือ
 *       hidden ด้วย md:hidden — Sidebar จะเข้ามาแทนตั้งแต่ lg ขึ้นไป
 * How:  pb-[max(0.5rem,env(safe-area-inset-bottom))] — รองรับ iPhone home indicator/notch
 *       icon FILL style เปลี่ยนเฉพาะ active item — บอกสถานะชัดเจนโดยไม่ต้องใช้สีพื้นหลัง
 */

import Link from "next/link";
import { usePathname } from "next/navigation";

// exact: true  → pathname === href        (สำหรับ "/" และหน้าที่มี path เดียว)
// exact: false → pathname.startsWith(href)  (ทำให้ /orders/abc ยัง highlight "คำสั่งซื้อ" อยู่)
const items = [
  { href: "/", icon: "home", label: "หน้าหลัก", exact: true },
  { href: "/products", icon: "storefront", label: "สินค้า", exact: false },
  { href: "/cart", icon: "shopping_cart", label: "ตะกร้า", exact: true },
  { href: "/orders", icon: "receipt_long", label: "คำสั่งซื้อ", exact: false },
  { href: "/profile", icon: "account_circle", label: "บัญชี", exact: false },
];

export default function MobileNav() {
  const pathname = usePathname();

  return (
    <nav className="md:hidden fixed bottom-0 w-full bg-surface-container-lowest/95 backdrop-blur-lg flex justify-around items-center py-2 pb-[max(0.5rem,env(safe-area-inset-bottom))] z-50 border-t border-outline-variant/10">
      {items.map((item) => {
        const isActive = item.exact ? pathname === item.href : pathname.startsWith(item.href);
        return (
          <Link
            key={item.href}
            href={item.href}
            className={`flex flex-col items-center gap-0.5 px-3 py-1 rounded-xl transition-colors ${isActive ? "text-primary" : "text-on-surface-variant"}`}
          >
            <span
              className={`material-symbols-outlined text-[22px] ${isActive ? "" : ""}`}
              style={isActive ? { fontVariationSettings: "'FILL' 1" } : undefined}
            >
              {item.icon}
            </span>
            <span className={`text-[10px] ${isActive ? "font-bold" : ""}`}>{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
