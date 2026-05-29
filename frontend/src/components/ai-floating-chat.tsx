"use client";

/**
 * AIFloatingChat — ปุ่มลอยมุมล่างขวาสำหรับเปิด AI chat
 *
 * What: Fixed button ที่ลิงก์ไปหน้า /chat ปรากฏในทุกหน้าของ shop
 * Why:  ให้ user เข้าถึง AI assistant ได้ตลอดเวลาโดยไม่ต้อง navigate ก่อน
 * How:  ใช้ CSS max-width transition สำหรับ expand label เมื่อ hover
 *       (max-w-0 → max-w-[10rem]) แทน opacity เพราะ text จะไม่ดัน layout กระตุก
 *       z-[60] สูงกว่า MobileNav (z-50) เพื่อให้ปุ่มอยู่บนสุดเสมอ
 */

import Link from "next/link";

export default function AIFloatingChat() {
  return (
    <div className="fixed bottom-8 right-8 z-[60] max-md:bottom-20 max-md:right-5">
      <Link
        href="/chat"
        className="group flex items-center gap-0 bg-white p-1.5 pl-1.5 rounded-full shadow-xl shadow-primary/15 border border-primary/10 hover:shadow-2xl hover:shadow-primary/25 transition-all"
      >
        <span className="max-w-0 overflow-hidden group-hover:max-w-[10rem] group-hover:pl-3 group-hover:pr-2 transition-all duration-300 ease-out text-sm font-bold text-primary whitespace-nowrap">
          ถาม AI ของเรา
        </span>
        <span className="w-12 h-12 editorial-gradient text-white rounded-full flex items-center justify-center">
          <span className="material-symbols-outlined text-2xl" style={{ fontVariationSettings: "'FILL' 1" }}>
            smart_toy
          </span>
        </span>
      </Link>
    </div>
  );
}
