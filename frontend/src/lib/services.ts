/**
 * services.ts — API service layer
 *
 * What: รวมฟังก์ชันเรียก API สำหรับแต่ละส่วนของแอป (auth, product, cart, order ฯลฯ)
 * Why:  จัดระเบียบให้ component ไม่ต้องรู้ endpoint URL เลย ใช้แค่ serviceX.method()
 *       ถ้าเปลี่ยน URL หรือ HTTP method แก้ที่ไฟล์นี้จุดเดียว
 * How:  แต่ละ service object ใช้ `api` helper จาก api.ts
 *       แบ่งตาม domain: authService, productService, cartService, ฯลฯ
 */
import { api } from "./api";
import type {
  LoginRequest,
  LoginAdminRequest,
  LoginResponse,
  LogoutRequest,
  RegisterRequest,
  UserProfile,
  AdminProfile,
  AuthUser,
  UpdateProfileRequest,
  ChangePasswordRequest,
  ProductListResponse,
  Product,
  Category,
  Attribute,
  AttributeValue,
  Cart,
  AddCartItemRequest,
  UpdateCartItemRequest,
  Order,
  CreateOrderRequest,
  CreateOrderResponse,
  ProcessPaymentRequest,
  PaymentResponse,
  OrderHistory,
  OrderHistoryListResponse,
  CatalogListResponse,
  CatalogProduct,
  CatalogVariantInfo,
  CreateProductRequest,
  AddVariantRequest,
  AdjustStockRequest,
  UpdateProductGeneralInfoRequest,
  UpdateVariantPriceRequest,
  SetProductActiveRequest,
  SetVariantActiveRequest,
  UpdateProductImagesRequest,
  UpdateVariantImagesRequest,
  ListProductRequest,
  CreateAdminRequest,
  ChatRequest,
  ChatResponse,
  AdminStats,
} from "./types";

/* ─── Auth ───
 * What: จัดการ authentication lifecycle ทั้งหมด
 * Why:  รวม endpoint เกี่ยวกับ user identity ไว้ที่เดียว
 * How:  login → รับ token, me → ดึงโปรไฟล์ปัจจุบัน, logout → ปิด session ฝั่ง server
 *       googleLogin → ใช้ Google ID Token แลกเอา JWT ของแอป
 */
export const authService = {
  login: (data: LoginRequest) =>
    api.post<LoginResponse>("/api/auth/auth/user-login", data),

  register: (data: RegisterRequest) =>
    api.post<UserProfile>("/api/auth/users/register", data),

  logout: () => {
    // What: ไม่ต้องส่ง refresh_token ใน body อีกต่อไป
    // Why:  backend อ่าน refresh_token จาก HttpOnly cookie อัตโนมัติ (Approach B)
    //       browser ส่ง cookie ให้เองเพราะเป็น same-origin request
    return api.post("/api/auth/auth/logout", {});
  },

  me: () => api.get<AuthUser>("/api/auth/auth/me"),

  getUser: (id: number) => api.get<UserProfile>(`/api/auth/users/${id}`),

  updateProfile: (id: number, data: UpdateProfileRequest) =>
    api.post<{ status: string; data: UserProfile }>(`/api/auth/users/update/${id}`, data),

  changePassword: (id: number, data: ChangePasswordRequest) =>
    api.post<{ message: string }>(`/api/auth/users/chgpass/${id}`, data),

  googleLogin: (idToken: string) =>
    api.post<LoginResponse>("/api/auth/auth/google", { id_token: idToken }),
};

/* ─── Products ───
 * What: ดึงข้อมูลสินค้าจาก product_service (หน้า admin-facing)
 * Why:  ใช้แยกต่างจาก catalogService เพราะคนละ service ทำหน้าที่ต่างกัน
 * How:  list() สร้าง query string จาก params object แล้วเรียก GET → ProductListResponse
 */
export const productService = {
  list: (params: ListProductRequest = {}) => {
    // params: { page: 2, search: "shirt", category: 3, is_active: true }
    //   → GET /api/products?page=2&search=shirt&category=3&is_active=true
    //
    // params: {}  (ทุก field optional — backend default: page=1, limit=10)
    //   → GET /api/products
    const q = new URLSearchParams();
    if (params.page)     q.set("page",     String(params.page));
    if (params.limit)    q.set("limit",    String(params.limit));
    if (params.search)   q.set("search",   params.search);
    if (params.category) q.set("category", String(params.category));
    if (params.is_active !== undefined) q.set("is_active", String(params.is_active));
    if (params.sort_by)  q.set("sort_by",  params.sort_by);
    if (params.order)    q.set("order",    params.order);
    return api.get<ProductListResponse>(`/api/products?${q.toString()}`);
  },

  get: (id: number) => api.get<Product>(`/api/products/${id}`),
};

/* ─── Categories ───
 * What: ดึงหมวดหมู่สินค้า
 * How:  list() คืน Category tree ทั้งหมด (รวม children แบบ nested อยู่แล้วจาก backend)
 */
export const categoryService = {
  list: () => api.get<Category[]>("/api/categories"),

  get: (id: number) => api.get<Category>(`/api/categories/${id}`),
};

/* ─── Attributes ───
 * What: ดึงรายการ attribute ทั้งหมด (เช่น Color, Size)
 * Why:  ใช้สร้าง dropdown ตอนเพิ่ม/แก้ไขสินค้า variant ในหน้า admin
 */
export const attributeService = {
  list: () => api.get<Attribute[]>("/api/attributes"),
};

/* ─── Cart ───
 * What: จัดการตะกร้าสินค้า (CRUD)
 * Why:  cart ถูกเก็บฝั่ง server เพื่อให้ sync ข้ามอุปกรณ์ได้
 * How:  updateItem/removeItem ใช้ variantId เป็น key (ไม่ใช่ cartItemId)
 *       ทุก method คืน Cart ออกมาเสมอ เพื่ออัปเดต context ให้ได้ทันที
 */
export const cartService = {
  get: () => api.get<Cart>("/api/cart"),

  addItem: (data: AddCartItemRequest) =>
    api.post<Cart>("/api/cart/items", data),

  updateItem: (variantId: number, data: UpdateCartItemRequest) =>
    api.put<Cart>(`/api/cart/items/${variantId}`, data),

  removeItem: (variantId: number) =>
    api.delete<Cart>(`/api/cart/items/${variantId}`),

  clear: () => api.delete<void>("/api/cart"),
};

/* ─── Orders (write operations) ───
 * What: สร้าง order และจัดการสถานะของ order (write path)
 * Why:  แยกออกจาก orderHistoryService เพราะ write ไป PostgreSQL, read ไป MongoDB
 * How:  create() → สร้าง order ใหม่
 *       cancel() → ยกเลิกพร้อมเหตุผล
 *       processPayment() → บันทึกการชำระเงิน (ผ่าน payment gateway token)
 */
export const orderService = {
  create: (data: CreateOrderRequest) =>
    api.post<CreateOrderResponse>("/api/orders", data),

  get: (id: string) => api.get<Order>(`/api/orders/${id}`),

  cancel: (id: string, reason: string) =>
    api.post<{ message: string }>(`/api/orders/${id}/cancel`, { reason }),

  processPayment: (id: string, data: ProcessPaymentRequest) =>
    api.post<{ message: string; payment: PaymentResponse }>(`/api/orders/${id}/pay`, data),
};

/* ─── Order History (read operations from MongoDB) ───
 * What: ดึงประวัติคำสั่งซื้อสำหรับแสดงผลในหน้า "Orders" ของ user
 * Why:  query จาก MongoDB ที่ denormalized เร็วกว่า join หลาย table ใน PostgreSQL
 * How:  list() รองรับ filter ตาม status และทำ pagination
 *       get() ดึงด้วย orderId (เป็น string UUID)
 */
export const orderHistoryService = {
  list: (page = 1, limit = 20, status?: string) => {
    // list(2, 20, "PAID")
    //   → GET /api/order-history?page=2&limit=20&status=PAID
    //
    // list()  (default)
    //   → GET /api/order-history?page=1&limit=20  (ไม่มี status = ดึงทุกสถานะ)
    const qs = new URLSearchParams();
    qs.set("page", String(page));
    qs.set("limit", String(limit));
    if (status) qs.set("status", status);
    return api.get<OrderHistoryListResponse>(`/api/order-history?${qs}`);
  },

  get: (orderId: string) =>
    api.get<OrderHistory>(`/api/order-history/${orderId}`),
};

/* ─── Catalog (search) ───
 * What: ค้นหาและ browse สินค้าสำหรับหน้า shop (read-optimized)
 * Why:  ใช้ catalog_service ที่ denormalized ไว้ใน search-friendly store
 *       แยกไว้ต่างหาก productService เพื่อไม่ผูกมัด URL กัน
 * How:  search() รองรับ filter/sort/pagination
 *       getVariant() ใช้ตอนแสดงรายละเอียดไอเทมในตะกร้า
 */
export const catalogService = {
  search: (params: {
    page?: number;
    limit?: number;
    search?: string;
    category_id?: number;
    sort_by?: string;
    order?: string;
  }) => {
    // { page: 1, search: "blue shirt", category_id: 5, sort_by: "price", order: "asc" }
    //   → GET /api/catalog/products?page=1&search=blue+shirt&category_id=5&sort_by=price&order=asc
    //
    // {}  → GET /api/catalog/products  (backend defaults apply)
    const qs = new URLSearchParams();
    if (params.page) qs.set("page", String(params.page));
    if (params.limit) qs.set("limit", String(params.limit));
    if (params.search) qs.set("search", params.search);
    if (params.category_id) qs.set("category_id", String(params.category_id));
    if (params.sort_by) qs.set("sort_by", params.sort_by);
    if (params.order) qs.set("order", params.order);
    return api.get<CatalogListResponse>(`/api/catalog/products?${qs}`);
  },

  get: (id: string) =>
    api.get<CatalogProduct>(`/api/catalog/products/${id}`),

  getVariant: (variantId: number) =>
    api.get<CatalogVariantInfo>(`/api/catalog/variants/${variantId}`),
};

/* ─── AI Chat ───
 * What: ส่ง message เข้า ai_service
 * Why:  ai_service แยกออกเป็น Python microservice เพราะใช้ LLM (ส่งผ่าน BFF ด้วย gRPC)
 * How:  BFF รับ POST /chat แล้ว forward ไป ai_service ผ่าน gRPC
 *       response มี reply (text) + products (สินค้าที่เกี่ยวข้อง)
 */
export const chatService = {
  send: (data: ChatRequest) =>
    api.post<ChatResponse>("/chat", data),
};

/* ─── Admin Auth ───
 * What: login/register สำหรับ admin
 * Why:  แยกจาก authService เพราะ endpoint ต่างกันและต้องใช้ X-Admin-Secret header
 * How:  register ต้องแนบ adminSecret ใน header เพื่อป้องกัน unauthorized admin creation
 */
export const adminAuthService = {
  login: (data: LoginAdminRequest) =>
    api.post<LoginResponse>("/api/auth/auth/admin-login", data),

  register: (data: CreateAdminRequest, adminSecret: string) =>
    api.post<AdminProfile>("/api/auth/admin/register", data, {
      headers: {
        "X-Admin-Secret": adminSecret,
      },
    }),
};

/* ─── Admin Products ───
 * What: CRUD และ manage stock/status สำหรับ admin
 * Why:  ทุก endpoint ใช้ /admin path ซึ่ง BFF ตรวจ admin role ก่อน forward
 * How:  ใช้ PATCH เพื่ออัปเดตบางส่วน (ราคา, stock, status, รูป)
 *       ใช้ PUT เพื่ออัปเดต general info (ชื่อ, description, category)
 */
export const adminProductService = {
  create: (data: CreateProductRequest) =>
    api.post<{ message: string }>("/api/products/admin", data),

  delete: (id: number) =>
    api.delete<void>(`/api/products/admin/${id}`),

  updateGeneralInfo: (id: number, data: UpdateProductGeneralInfoRequest) =>
    api.put<void>(`/api/products/admin/${id}/general-info`, data),

  updateVariantPrice: (productId: number, variantId: number, data: UpdateVariantPriceRequest) =>
    api.patch<void>(`/api/products/admin/${productId}/variants/${variantId}/price`, data),

  adjustStock: (productId: number, variantId: number, data: AdjustStockRequest) =>
    api.patch<void>(`/api/products/admin/${productId}/variants/${variantId}/stock`, data),

  addVariant: (productId: number, data: AddVariantRequest) =>
    api.post<void>(`/api/products/admin/${productId}/variants`, data),

  setProductActive: (productId: number, data: SetProductActiveRequest) =>
    api.patch<void>(`/api/products/admin/${productId}/active`, data),

  setVariantActive: (productId: number, variantId: number, data: SetVariantActiveRequest) =>
    api.patch<void>(`/api/products/admin/${productId}/variants/${variantId}/active`, data),

  updateProductImages: (productId: number, data: UpdateProductImagesRequest) =>
    api.patch<void>(`/api/products/admin/${productId}/images`, data),

  updateVariantImages: (productId: number, variantId: number, data: UpdateVariantImagesRequest) =>
    api.patch<void>(`/api/products/admin/${productId}/variants/${variantId}/images`, data),
};

/* ─── Admin Attributes ───
 * What: CRUD attribute และค่าของ attribute (color values, size values เป็นต้น)
 * Why:  admin ต้องกำหนด attribute ก่อนจึงสร้างสินค้า variant ได้
 */
export const adminAttributeService = {
  create: (data: { name: string }) =>
    api.post<Attribute>("/api/attributes/admin", data),

  update: (id: number, data: { name: string }) =>
    api.put<void>(`/api/attributes/admin/${id}`, data),

  createValue: (attributeId: number, data: { value: string }) =>
    api.post<AttributeValue>(`/api/attributes/admin/${attributeId}/values`, data),

  deleteValue: (attributeId: number, valueId: number) =>
    api.delete<void>(`/api/attributes/admin/${attributeId}/values/${valueId}`),

  delete: (id: number) =>
    api.delete<void>(`/api/attributes/admin/${id}`),
};

/* ─── Admin Categories ───
 * What: CRUD หมวดหมู่สินค้าและ toggle active
 * Why:  admin ต้องจัดโครงสร้าง category ก่อนผูกสินค้า
 * How:  toggleActive ส่ง is_active ใน body เพื่อ set ค่าตรงๆ
 */
export const adminCategoryService = {
  create: (data: { name: string; slug: string; is_active: boolean; description?: string; parent_id?: number }) =>
    api.post<Category>("/api/categories/admin", data),

  update: (id: number, data: { name: string; slug: string; is_active: boolean; description?: string; parent_id?: number }) =>
    api.put<Category>(`/api/categories/admin/${id}`, data),

  delete: (id: number) =>
    api.delete<void>(`/api/categories/admin/${id}`),

  toggleActive: (id: number, isActive: boolean) =>
    api.patch<void>(`/api/categories/admin/${id}/active`, { is_active: isActive }),
};

/* ─── Admin Orders ───
 * What: admin action เกี่ยวกับ order
 * Why:  admin ยกเลิก order ได้จากทุกสถานะ (ไม่จำกัดเหมือน user)
 */
export const adminOrderService = {
  cancel: (id: string, reason: string) =>
    api.post<{ message: string }>(`/api/orders/admin/${id}/cancel`, { reason }),
};

/* ─── Admin Order History ───
 * What: ดู order ทั้งหมดในระบบสำหรับ admin (ต่างจาก orderHistoryService ซึ่ง filter ตาม user)
 * Why:  orderHistoryService.list() return เฉพาะ order ของ user คนนั้น
 *       admin ต้องเห็นทุก order → ใช้ endpoint แยก /api/order-history/admin
 */
export const adminOrderHistoryService = {
  list: (page = 1, limit = 10, status?: string) => {
    const qs = new URLSearchParams();
    qs.set("page", String(page));
    qs.set("limit", String(limit));
    if (status) qs.set("status", status);
    return api.get<OrderHistoryListResponse>(`/api/order-history/admin?${qs}`);
  },

  get: (orderId: string) =>
    api.get<OrderHistory>(`/api/order-history/admin/${orderId}`),

  stats: () =>
    api.get<AdminStats>(`/api/order-history/admin/stats`),
};
