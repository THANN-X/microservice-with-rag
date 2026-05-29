/**
 * types.ts — Type definitions สำหรับทั้งแอป
 *
 * What: รวม TypeScript interface/type ทุกตัวที่ใช้ในแอป (request body, response, สถานะ)
 * Why:  ให้ TypeScript ตรวจสอบ type ได้ตั้งแต่ compile-time ลดโอกาส runtime error
 *       และเป็น single source of truth ของ data contract ระหว่าง frontend กับ backend
 * How:  แต่ละ interface map 1:1 กับ Go DTO ของ backend ที่ส่งผ่าน BFF → nginx
 *       เมื่อ backend เปลี่ยน DTO ให้มาแก้ที่ไฟล์นี้จุดเดียว
 */

// ─── Auth ───
// ครอบคลุม request/response ที่เกี่ยวกับการ login, register, token และข้อมูลผู้ใช้
export interface LoginRequest {
  email: string;
  password: string;
  device_info?: string;
}

export interface LoginAdminRequest {
  username: string;
  password: string;
  device_info?: string;
}

export interface LoginResponse {
  access_token: string;
  // Why: refresh_token ไม่อยู่ใน body อีกต่อไป — อยู่ใน HttpOnly cookie แทน (Approach B)
  refresh_token?: string;
}

export interface RefreshRequest {
  refresh_token?: string;
}

export interface LogoutRequest {
  // Why: refresh_token อ่านจาก HttpOnly cookie ที่ backend ไม่ต้องส่งใน body
  refresh_token?: string;
}

export interface RegisterRequest {
  first_name: string;
  last_name: string;
  email: string;
  password: string;
  phone: string;
  address: string;
}

// UserProfile — regular user (email-based login)
export interface UserProfile {
  id: number;
  role: "user";
  first_name: string;
  last_name: string;
  email: string;
  phone: string;
  address: string;
}

// AdminProfile — admin account (username-based login)
export interface AdminProfile {
  id: number;
  role: "admin";
  first_name: string;
  last_name: string;
  username: string;
  phone: string;
  address: string;
}

// AuthUser — discriminated union ใช้ใน AuthContext
// ตรวจ role ด้วย user.role === "admin" เพื่อ narrow type
export type AuthUser = UserProfile | AdminProfile;

export interface UpdateProfileRequest {
  first_name: string;
  last_name: string;
  phone: string;
  address: string;
}

export type UpdateUserRequest = UpdateProfileRequest;

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

// ─── Product ───
// โครงสร้างสินค้า — Product มีหลาย Variant, แต่ละ Variant มีหลาย VariantOption
// เป็นโครงสร้างแบบ nested ที่รองรับ attribute เช่น Color: Red, Size: M
export interface Product {
  id: number;
  name: string;
  description: string;
  image_urls: string[];
  is_active: boolean;
  variants: Variant[];
  categories: ProductCategory[];
  created_by: number;
}

export interface ProductCategory {
  id: number;
  name: string;
}

export interface Variant {
  id: number;
  sku: string;
  name: string;
  price: number;
  stock: number;
  is_active: boolean;
  image_urls: string[];
  options: VariantOption[];
}

export interface VariantOption {
  name: string;  // e.g. "Color"
  value: string; // e.g. "Red"
}

export interface ProductListResponse {
  items: Product[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// ─── Category ───
// tree โครงสร้างหมวดหมู่สินค้าแบบ nested — มี parent_id และ children[] สำหรับเมนูแบบ dropdown ซ้อนกัน
export interface Category {
  id: number;
  name: string;
  slug: string;
  is_active: boolean;
  parent_id: number | null;
  children: Category[];
}

// ─── Attribute ───
// attribute ของสินค้า เช่น Attribute = { name: "Color", values: ["Red", "Blue"] }
// ใช้ในหน้า admin สร้าง/แก้ไข variant option ของสินค้า
export interface Attribute {
  id: number;
  name: string;
  values: AttributeValue[];
}

export interface AttributeValue {
  id: number;
  value: string;
}

// ─── Cart ───
// ตะกร้าสินค้าสำหรับ user คนเดียว
// CartItem.variant_id คือ key หลักตอนสร้าง order — field ที่มี ? optional คือข้อมูลที่ join มาสำหรับแสดผล
export interface Cart {
  id: number;
  user_id: number;
  items: CartItem[];
  created_at: string;
  updated_at: string;
}

export interface CartItem {
  id: number;
  variant_id: number;
  quantity: number;
  variant_name?: string;
  product_name?: string;
  price?: number;
  image_url?: string;
}

export interface AddCartItemRequest {
  variant_id: number;
  quantity: number;
}

export interface UpdateCartItemRequest {
  quantity: number;
}

// ─── Order ───
// คำสั่งซื้อ — write path ไปที่ order_service (PostgreSQL)
// OrderStatus เป็น finite state machine: PENDING → CONFIRMED → AWAITING_PAYMENT → PAID → COMPLETED
// หรือ CANCELLED จากทุกสถานะก่อน PAID
export interface Order {
  id: string;
  customer_id: number;
  status: OrderStatus;
  total_amount: number;
  items: OrderItem[];
  shipping_address: ShippingAddress;
  note: string;
  created_at: string;
  updated_at: string;
}

export type OrderStatus = "PENDING" | "CONFIRMED" | "AWAITING_PAYMENT" | "PAID" | "COMPLETED" | "CANCELLED";

export interface OrderItem {
  id: string;
  variant_id: number;
  quantity: number;
  unit_price: number;
  subtotal: number;
}

export interface ShippingAddress {
  full_name: string;
  phone: string;
  address_line: string;
  sub_district: string;
  district: string;
  province: string;
  postal_code: string;
}

export interface CreateOrderRequest {
  items: { variant_id: number; quantity: number; unit_price: number }[];
  shipping_address: ShippingAddress;
  note?: string;
}

export interface CreateOrderResponse {
  message: string;
  order: Order;
}

export interface ProcessPaymentRequest {
  token: string;
  payment_method: string;
}

export interface PaymentResponse {
  id: string;
  order_id: string;
  amount: number;
  currency: string;
  status: string;
  gateway: string;
  gateway_charge_id?: string;
  payment_method: string;
  paid_at?: string;
  created_at: string;
}

// ─── Order History (read-optimized from MongoDB) ───
// ข้อมูลเดียวกับ Order แต่อ่านจาก order_history_service ที่ใช้ MongoDB
// Why: MongoDB read เร็วกว่า PostgreSQL join สำหรับ query history ที่มีปริมาณมาก
// ข้อมูลถูก sync มาจาก order_service ผ่าน event (CQRS pattern)
export interface OrderHistory {
  order_id: string;
  customer_id: number;
  status: string;
  total_amount: number;
  items: OrderHistoryItem[];
  shipping_address: ShippingAddress;
  note: string;
  cancel_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface OrderHistoryItem {
  variant_id: number;
  quantity: number;
  unit_price: number;
  subtotal: number;
}

export interface OrderHistoryListResponse {
  items: OrderHistory[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface AdminStats {
  total_orders: number;
  total_revenue: number;
}

// ─── Catalog (read-optimized) ───
// ข้อมูลสินค้าสำหรับหน้า shop/search — อ่านจาก catalog_service ที่ sync สินค้าจาก product_service
// CatalogProduct ใช้ id เป็น string (เป็น MongoDB ObjectID) ไม่ใช่ number
export interface CatalogProduct {
  id: string;
  product_id: number;
  name: string;
  description: string;
  image_urls: string[];
  variants: Variant[];
  categories: Category[];
}

export interface CatalogListResponse {
  products: CatalogProduct[];
  total: number;
  page: number;
  limit: number;
}

export interface CatalogVariantInfo {
  product_name: string;
  variant_name: string;
  price: number;
  image_url: string;
  image_urls: string[];
}

// ─── Admin ───
// Request type สำหรับหน้า admin panel โดยเฉพาะ — อาศัย X-Admin-Secret header (สอนใน api.ts)
// แยก type ออกมาต่างหากปกติ เพื่อให้ชัดเจนว่าอะไร admin เท่านั้น vs ทุกคน
export interface CreateProductRequest {
  name: string;
  description: string;
  image_urls: string[];
  variants: CreateVariantRequest[];
  category_ids: number[];
}

export interface CreateVariantRequest {
  sku: string;
  name: string;
  price: number;
  stock: number;
  attribute_value_ids: number[];
}

export interface AddVariantRequest {
  sku: string;
  name: string;
  price: number;
  stock: number;
  attribute_value_ids: number[];
}

export interface AdjustStockRequest {
  new_stock: number;
  reason: string;
}

export interface UpdateProductGeneralInfoRequest {
  name: string;
  description: string;
  category_ids: number[];
}

export interface UpdateVariantPriceRequest {
  new_price: number;
}

export interface SetProductActiveRequest {
  is_active: boolean;
}

export interface SetVariantActiveRequest {
  is_active: boolean;
}

export interface UpdateProductImagesRequest {
  image_urls: string[];
}

export interface UpdateVariantImagesRequest {
  image_urls: string[];
}

export interface ListProductRequest {
  page?: number;
  limit?: number;
  search?: string;
  category?: number;
  is_active?: boolean;
  sort_by?: "created_at" | "name" | "price" | "stock";
  order?: "asc" | "desc";
}

export interface CreateAdminRequest {
  first_name: string;
  last_name: string;
  username: string;
  password: string;
  phone: string;
  address: string;
}

export interface UpdateAdminRequest {
  first_name: string;
  last_name: string;
  phone: string;
  address: string;
}

// AdminProfile is defined in the Auth section above (UserProfile | AdminProfile)
export type AdminLoginRequest = LoginAdminRequest;

// ─── AI Chat ───
// ใช้ส่งเข้า ai_service โดยตรง (ผ่าน BFF /chat endpoint)
// ChatResponse อาจมี products แนบมากับ reply เพื่อให้สี floating chat แสดการ์ดสินค้าที่เกี่ยวข้อง
export interface ChatRequest {
  message: string;
  session_id?: string;
}

export interface ChatResponse {
  reply: string;
  products: ChatProductReference[];
}

export interface ChatProductReference {
  product_id: number;
  name: string;
  description: string;
  min_price: number;
  relevance_score: number;
}

