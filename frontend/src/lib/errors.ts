/**
 * errors.ts — แปลง error จากชั้น API ให้เป็นข้อความที่โชว์ผู้ใช้ได้
 *
 * What: ดึงข้อความ error ที่ "อ่านรู้เรื่อง" ออกมาโชว์ผู้ใช้
 * Why:  api.ts โยน Error ที่อาจเป็นข้อความจาก backend (ใช้ได้) หรือ
 *       เป็น string เชิงเทคนิคอย่าง "Request failed: 500" / "Unauthorized" (ไม่ควรโชว์)
 * How:  กรองตัวเชิงเทคนิคออกด้วย regex แล้ว fallback เป็นข้อความที่เราเขียนเองแทน
 */

/** pattern ของข้อความที่มาจากชั้น transport — ไม่มีความหมายกับผู้ใช้ปลายทาง */
const TECHNICAL_MESSAGE = /^(Request failed|Unauthorized|Failed to fetch|NetworkError)/i;

export function toUserMessage(err: unknown, fallback: string): string {
  if (!(err instanceof Error) || !err.message) return fallback;
  return TECHNICAL_MESSAGE.test(err.message) ? fallback : err.message;
}
