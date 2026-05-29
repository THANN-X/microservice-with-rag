"use client";

/**
 * Navbar — navigation bar หลักด้านบน (desktop + tablet)
 *
 * What: แสดง brand, nav links, search, cart badge และ user menu
 * Why:  fixed ด้านบนเสมอ เพื่อให้ user เข้าถึงได้ตลอดเวลา scroll
 * How:  อ่าน user จาก AuthContext → ตัดสิน render login button หรือ profile/logout
 *       อ่าน itemCount จาก CartContext → แสดง badge บนไอคอนตะกร้า
 *       handleSearch → encodeURIComponent แล้ว push ไป /products?search=...
 */

import Link from "next/link";
import { useState } from "react";
import { usePathname } from "next/navigation";
import { useAuth } from "@/context/auth-context";
import { useCart } from "@/context/cart-context";
import { useRouter } from "next/navigation";

// ประกาศนอก component — static reference ที่ไม่ถูกสร้างใหม่ทุก render
const navLinks = [
  { href: "/", label: "หน้าแรก" },
  { href: "/products", label: "สินค้า" },
  { href: "/orders", label: "คำสั่งซื้อ" },
];

export default function Navbar() {
  const { user, logout } = useAuth();
  const { itemCount } = useCart();
  const [query, setQuery] = useState("");
  const router = useRouter();
  const pathname = usePathname();

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    // query: "blue shirt"  →  router.push("/products?search=blue%20shirt")
    // encodeURIComponent จัดการ space, อักษรไทย และ special chars ให้ URL-safe
    if (query.trim()) {
      router.push(`/products?search=${encodeURIComponent(query.trim())}`);
    }
  };

  return (
    <nav className="fixed top-0 w-full z-50 glass-nav border-b border-surface-container/50 px-6 lg:px-8 h-16 flex justify-between items-center">
      {/* Brand */}
      <Link href="/" className="text-2xl font-black text-primary tracking-tight select-none">
        อนันตา
      </Link>

      {/* Nav Links - hidden on mobile */}
      <div className="hidden md:flex items-center gap-1">
        {navLinks.map((link) => {
          // "/" ต้องใช้ exact match — ถ้าใช้ startsWith ทุก path จะ active หมด
          // path อื่นใช้ startsWith — ทำให้ /products/123 ยัง highlight "สินค้า" อยู่
          const active = link.href === "/" ? pathname === "/" : pathname.startsWith(link.href);
          return (
            <Link
              key={link.href}
              href={link.href}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors duration-200 ${
                active
                  ? "text-primary bg-primary/5 font-bold"
                  : "text-on-surface-variant hover:text-primary hover:bg-primary/5"
              }`}
            >
              {link.label}
            </Link>
          );
        })}
      </div>

      {/* Right Actions */}
      <div className="flex items-center gap-2">
        {/* Search */}
        <form onSubmit={handleSearch} className="hidden lg:flex items-center bg-surface-container-highest/60 px-3 py-2 rounded-xl gap-2 focus-within:bg-surface-container-highest focus-within:ring-2 focus-within:ring-primary/10 transition-all">
          <span className="material-symbols-outlined text-[20px] text-outline">search</span>
          <input
            type="text"
            className="bg-transparent border-none focus:ring-0 text-sm w-44 outline-none placeholder:text-outline/60"
            placeholder="ค้นหาสินค้า..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </form>

        {/* Icon buttons */}
        <div className="flex items-center gap-1">
          <Link
            href="/cart"
            className="relative w-10 h-10 flex items-center justify-center rounded-xl text-on-surface-variant hover:bg-surface-container-low hover:text-primary transition-colors"
          >
            <span className="material-symbols-outlined text-[22px]">shopping_cart</span>
            {itemCount > 0 && (
              <span className="absolute top-1 right-1 min-w-[18px] h-[18px] bg-error text-on-error text-[10px] font-bold rounded-full flex items-center justify-center px-1 leading-none">
                {/* จำกัดแสดงสูงสุด "99+" — badge จะไม่ล้นเมื่อมีสินค้าเยอะ */}
                {itemCount > 99 ? "99+" : itemCount}
              </span>
            )}
          </Link>

          {/* user มีค่า (logged in) → แสดง profile + logout
               user เป็น null     → แสดงปุ่ม "เข้าสู่ระบบ" */}
          {user ? (
            <>
              <Link
                href="/profile"
                className="w-10 h-10 flex items-center justify-center rounded-xl text-on-surface-variant hover:bg-surface-container-low hover:text-primary transition-colors"
              >
                <span className="material-symbols-outlined text-[22px]">person</span>
              </Link>
              <button
                onClick={logout}
                className="w-10 h-10 flex items-center justify-center rounded-xl text-on-surface-variant/60 hover:bg-error/5 hover:text-error transition-colors"
                title="ออกจากระบบ"
              >
                <span className="material-symbols-outlined text-[20px]">logout</span>
              </button>
            </>
          ) : (
            <Link
              href="/login"
              className="ml-1 px-5 py-2 rounded-xl editorial-gradient text-white text-sm font-bold hover:shadow-md hover:shadow-primary/20 transition-shadow"
            >
              เข้าสู่ระบบ
            </Link>
          )}
        </div>
      </div>
    </nav>
  );
}
