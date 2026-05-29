"use client";

/**
 * Sidebar — navigation panel ด้านซ้าย (desktop only, lg ขึ้นไป)
 *
 * What: แสดง user profile, nav links และปุ่ม CTA ช้อปปิ้ง
 * Why:  desktop มีพื้นที่พอให้แสดง nav แบบ full-label ทางซ้ายได้
 *       mobile ใช้ MobileNav (ด้านล่าง) แทน— hidden ด้วย hidden lg:flex
 * How:  isActive() รับค่า exact flag เพราะ "/" จะ match ทุก path ถ้าใช้ startsWith
 *       avatar initial ใช้ optional chain (ป้องกัน crash ก่อนที่ user โหลดเสร็จ)
 */

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/context/auth-context";

// exact: true  → pathname === href        (ใช้สำหรับ "/" และหน้าที่มี path เดียว)
// exact: false → pathname.startsWith(href)  (ทำให้ /products/123 ยัง highlight "สินค้า" อยู่)
const navItems = [
  { href: "/", icon: "home", label: "หน้าหลัก", exact: true },
  { href: "/products", icon: "storefront", label: "สินค้า", exact: false },
  { href: "/orders", icon: "receipt_long", label: "คำสั่งซื้อ", exact: false },
  { href: "/coupons", icon: "confirmation_number", label: "คูปอง", exact: true },
  { href: "/chat", icon: "smart_toy", label: "แชทกับ AI", exact: true },
];

export default function Sidebar() {
  const pathname = usePathname();
  const { user } = useAuth();

  // แยก logic ออกมาเป็น function เพื่อให้ .map() อ่านง่ายขึ้น
  const isActive = (href: string, exact: boolean) =>
    exact ? pathname === href : pathname.startsWith(href);

  return (
    <aside className="fixed left-0 top-0 h-full w-64 z-40 bg-surface-container-lowest border-r border-surface-container hidden lg:flex flex-col pt-24 pb-8">
      {/* User Profile */}
      <div className="px-6 mb-8">
        <Link href="/profile" className="flex items-center gap-3 group">
          {/* avatar initial: user.first_name ตัวแรก uppercase
               optional chain (?.) ป้องกัน crash ถ้า user หรือ first_name ยังเป็น undefined
               fallback "?" แสดงเมื่อ guest (ยังไม่ login) */}
          <div className="w-11 h-11 rounded-full editorial-gradient flex items-center justify-center text-white font-bold text-sm shadow-md group-hover:shadow-lg transition-shadow">
            {user?.first_name?.charAt(0)?.toUpperCase() || "?"}
          </div>
          <div className="min-w-0">
            <p className="text-[11px] text-outline leading-none mb-1">
              {user ? "สวัสดี" : "ยินดีต้อนรับ"}
            </p>
            <p className="font-bold text-on-surface text-sm truncate">
              {user ? `${user.first_name} ${user.last_name}` : "เข้าสู่ระบบ"}
            </p>
          </div>
        </Link>
      </div>

      {/* Divider */}
      <div className="mx-6 border-t border-surface-container mb-4" />

      {/* Navigation */}
      <nav className="flex-1 px-3 space-y-1">
        {navItems.map((item) => {
          const active = isActive(item.href, item.exact);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 px-4 py-3 rounded-xl text-sm transition-all duration-200 ${
                active
                  ? "bg-primary text-white font-semibold shadow-md shadow-primary/20"
                  : "text-on-surface-variant hover:bg-surface-container-low hover:text-on-surface"
              }`}
            >
              <span
                className="material-symbols-outlined text-[20px]"
                style={active ? { fontVariationSettings: "'FILL' 1" } : undefined}
              >
                {item.icon}
              </span>
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>

      {/* Bottom CTA */}
      <div className="px-5 space-y-3">
        <div className="border-t border-surface-container mb-1" />
        <Link
          href="/products"
          className="flex items-center justify-center gap-2 w-full editorial-gradient text-white py-3 rounded-xl font-bold text-sm shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-shadow"
        >
          <span className="material-symbols-outlined text-[18px]">shopping_bag</span>
          เริ่มช้อปปิ้ง
        </Link>
      </div>
    </aside>
  );
}
