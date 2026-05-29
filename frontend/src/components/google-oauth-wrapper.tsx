"use client";

/**
 * GoogleOAuthWrapper — Provider wrapper สำหรับ Google OAuth
 *
 * What: ครอบส่วนของ app ที่ต้องการ Google login ด้วย GoogleOAuthProvider
 * Why:  @react-oauth/google ต้องการ context provider ครอบก่อนถึงจะใช้ useGoogleLogin() ได้
 *       แยกออกมาเป็น component เพื่อไม่ให้ layout หลักต้องรู้เรื่อง Google library โดยตรง
 * How:  อ่าน Client ID จาก env NEXT_PUBLIC_GOOGLE_CLIENT_ID
 *       ถ้าไม่มีค่า → fallback เป็น "" (Google SDK จะ log warning แต่ไม่ crash)
 */

import { GoogleOAuthProvider } from "@react-oauth/google";
import type { ReactNode } from "react";

// NEXT_PUBLIC_ prefix จำเป็น — Next.js expose ให้ browser เฉพาะ env ที่ขึ้นต้นด้วยนี้เท่านั้น
const GOOGLE_CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? "";

export default function GoogleOAuthWrapper({ children }: { children: ReactNode }) {
  return (
    <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>
      {children}
    </GoogleOAuthProvider>
  );
}
