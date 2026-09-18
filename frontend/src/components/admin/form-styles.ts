/**
 * form-styles.ts — class ของ input/label ที่ฟอร์ม admin ใช้ซ้ำ
 *
 * Why: class string ชุดนี้เคยถูก copy ไว้ ~15 จุดในหน้าเดียว พอจะปรับ focus ring
 *      หรือ radius ทีต้องไล่แก้ทุกจุดแล้วมักตกหล่น จนแต่ละช่องหน้าตาไม่ตรงกัน
 */

/** ช่องกรอกขนาดปกติ — ใช้ในฟอร์มหลักของ modal */
export const inputField =
  "w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20";

/** ช่องกรอกขนาดเล็ก บนพื้น surface — ใช้ในฟอร์มย่อยที่อยู่ในกรอบอีกที */
export const inputFieldSm =
  "w-full rounded-xl bg-surface-low/40 px-3 py-2 text-sm outline-none transition-all focus:ring-2 focus:ring-primary/20";

/** ช่องกรอกขนาดเล็ก บนพื้นขาว — ใช้ในการ์ด variant ซึ่งตัวการ์ดเป็นพื้น surface อยู่แล้ว */
export const inputFieldOnCard =
  "w-full rounded-xl bg-white px-3 py-2 text-sm outline-none transition-all focus:ring-2 focus:ring-primary/20";

export const fieldLabel = "mb-1 block text-xs font-medium text-secondary";
