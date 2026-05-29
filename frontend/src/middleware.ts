/**
 * middleware.ts — Next.js Edge Middleware สำหรับตรวจ auth ที่ฝั่ง server
 *
 * What: ตรวจ refresh_token cookie ก่อน render หน้า /admin/*
 * Why:  ป้องกัน FOUC (Flash of Unauthenticated Content) ที่เกิดจาก client-side auth check
 *       ถ้าไม่มี cookie เลย → redirect ไปที่ /admin/login ทันทีโดยไม่ต้องโหลดหน้า
 * Note: middleware ตรวจแค่ cookie มีอยู่หรือเปล่า (ไม่ verify JWT)
 *       การ verify จริงทำที่ BFF gateway auth middleware (Go/Fiber)
 */

import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // What: ตรวจเฉพาะ /admin/* ที่ไม่ใช่ login page
  // Why:  /admin/login ไม่ต้องการ cookie (มิฉะนั้น redirect วนไม่หยุด)
  if (pathname.startsWith("/admin") && pathname !== "/admin/login") {
    const refreshToken = request.cookies.get("refresh_token");

    if (!refreshToken) {
      const loginUrl =  new URL("/admin/login", request.url);
      return NextResponse.redirect(loginUrl);
    }
  }

  return NextResponse.next();
}

export const config = {
  // What: matcher กำหนดว่า middleware นี้ทำงานกับ path ไหนบ้าง
  // Why:  จำกัดเฉพาะ /admin/* เพื่อไม่ให้ middleware ทำงานกับทุก request
  matcher: ["/admin/:path*"],
};
