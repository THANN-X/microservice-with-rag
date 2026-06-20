/**
 * api.ts — HTTP client กลางของทั้งแอป
 *
 * What: wrapper บาง ๆ รอบ fetch() ที่ใช้ส่ง API request ทุกชนิด
 * Why:  รวม logic ซ้ำ ๆ ไว้ที่เดียว เช่น inject JWT, handle 401, parse error
 *       service layer จะได้เขียนแค่ endpoint + body โดยไม่ต้องสนใจ auth ทุกครั้ง
 * How:  export object `api` ที่มี .get / .post / .put / .patch / .delete
 *       ทุก method เรียกใช้ request() ภายในซึ่ง handle token และ error กลาง
 *
 * Security (Approach B):
 *   - access_token เก็บใน JS module variable (ไม่ใช้ localStorage → ป้องกัน XSS อ่านข้าม session)
 *   - refresh_token อยู่ใน HttpOnly cookie (JS อ่านไม่ได้ → ป้องกัน XSS ขโมย token)
 *   - Browser ส่ง cookie อัตโนมัติ Next.js rewrite proxy forward ให้ BFF → auth-service
 */

// Base URL prefix — ใช้ empty string เพราะ Next.js rewrite proxy ต่อ path ให้เองอยู่แล้ว
const BASE = "";

// What: access_token เก็บใน module-level variable แทน localStorage
// Why:  localStorage อ่านได้จาก JS ทุก script → เสี่ยง XSS
//       memory variable อ่านได้เฉพาะ code ในแอปนี้ และหายไปเมื่อ page reload (ปลอดภัยกว่า)
let accessToken: string | null = null;

/**
 * What: ตั้งค่า access_token ใน memory (เรียกหลัง login หรือ refresh สำเร็จ)
 */
export function setAccessToken(token: string | null): void {
  accessToken = token;
}

/**
 * What: ฟังก์ชันกลางสำหรับส่ง HTTP request ทุกชนิดในแอปนี้
 * Why:  รวม inject JWT, handle 401+retry, error parsing ไว้ที่เดียว
 * How:  ลำดับขั้นตอน:
 *         1) ดึง access_token จาก module variable (ถ้าอยู่ใน browser)
 *         2) แนบ Authorization: Bearer <token> ใน header ทุก request
 *         3) เรียก fetch() → ถ้า 401 (token หมดอายุ) ลอง tryRefresh() แล้ว retry ครั้งเดียว
 *         4) ถ้า refresh ล้มเหลว → throw error ให้ caller (เช่น auth-context) จัดการ redirect
 *         5) ถ้า response ไม่ ok → parse body แล้ว throw error พร้อม message จาก server
 *         6) 204 No Content → return empty object (ไม่มี body ให้ parse)
 */
async function request<T>(
  url: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...options.headers,
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  };

  // What: credentials: "include" บังคับให้ browser แนบ cookie ทุก request
  // Why:  แม้ same-origin แต่ Fetch API spec ไม่รับประกัน cookie ใน Safari / edge cases
  //       ระบุชัดเจนเพื่อให้ refresh_token cookie ถูกส่งไปเสมอ
  const res = await fetch(`${BASE}${url}`, { ...options, headers, credentials: "include" });

  if (res.status === 401 && typeof window !== "undefined") {
    const refreshed = await tryRefresh();
    if (refreshed) {
      return request<T>(url, options);
    }
    // What: refresh ล้มเหลว → ล้าง token ที่หมดอายุออกจาก memory
    // Why:  ไม่ redirect ตรงนี้ — ให้ caller (auth-context / component) จัดการ redirect เอง
    //       เพราะ admin ควรไปที่ /admin/login ส่วน user ควรไปที่ /login
    setAccessToken(null);
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    const backendError = body.error || body.message || body.details;
  
    throw new Error(backendError ?? `Request failed: ${res.status}`);
  }

  if (res.status === 204) return {} as T;
  return res.json();
}

/**
 * What: พยายามต่ออายุ access_token โดยใช้ refresh_token ที่อยู่ใน HttpOnly cookie
 * Why:  access_token มีอายุสั้น (เช่น 15 นาที) เพื่อความปลอดภัย
 *       แต่ user ไม่ควรต้อง login ใหม่ทุกครั้งที่ token หมดอายุ
 * How:  1) POST ไปที่ endpoint refresh-token — browser ส่ง cookie อัตโนมัติ (same-origin)
 *       2) Next.js rewrite proxy forward Cookie header ไปยัง BFF → auth-service
 *       3) auth-service อ่าน refresh_token จาก cookie → ออก access_token ใหม่
 *       4) ถ้าสำเร็จ → อัปเดต accessToken ใน memory แล้ว return true
 *       5) ถ้าล้มเหลว → return false ให้ request() throw Unauthorized
 */
async function tryRefresh(): Promise<boolean> {
  try {
    const res = await fetch(`${BASE}/api/auth/auth/refresh-token`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      // What: ส่ง body เป็น {} เพื่อให้ Fiber BodyParser ได้ valid JSON
      // Why:  ถ้าส่ง body ว่างพร้อม Content-Type: application/json
      //       Fiber จะ fail "unexpected end of JSON input" → BodyParser error → 400
      //       ส่ง {} แทน → parse สำเร็จ refresh_token เป็น empty → fallback อ่าน cookie
      body: JSON.stringify({}),
      credentials: "include",
    });
    if (!res.ok) return false;
    const data = await res.json();
    if (!data.access_token) return false;
    // What: อัปเดต in-memory access_token
    // Why:  cookie ใหม่ (refresh_token rotation) ถูก set โดย browser อัตโนมัติจาก Set-Cookie header
    setAccessToken(data.access_token);
    return true;
  } catch {
    return false;
  }
}

/**
 * What: HTTP method shortcuts ที่ export ออกไปให้ service layer ใช้
 * Why:  ทำให้โค้ดใน services.ts กระชับ เช่น api.get<User>(url)
 * How:  แต่ละ key ก็แค่ wrap request() พร้อมกำหนด method + JSON.stringify body
 */
export const api = {
  get: <T>(url: string) => request<T>(url),
  post: <T>(url: string, body?: unknown, options: RequestInit = {}) =>
    request<T>(url, { ...options, method: "POST", body: JSON.stringify(body) }),
  put: <T>(url: string, body?: unknown, options: RequestInit = {}) =>
    request<T>(url, { ...options, method: "PUT", body: JSON.stringify(body) }),
  patch: <T>(url: string, body?: unknown, options: RequestInit = {}) =>
    request<T>(url, { ...options, method: "PATCH", body: JSON.stringify(body) }),
  delete: <T>(url: string, options: RequestInit = {}) =>
    request<T>(url, { ...options, method: "DELETE" }),
};
