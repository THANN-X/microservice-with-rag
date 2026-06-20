/**
 * server-fetch.ts — Server-side fetch utilities สำหรับ public API
 *
 * Why: api.ts ใช้ browser-specific features (in-memory accessToken, credentials: "include")
 *      Server Components ไม่มี browser context → ใช้ api.ts ไม่ได้
 *      Next.js rewrites ใช้ได้เฉพาะ browser request เท่านั้น → SC ต้อง fetch ตรงไป BFF
 *      Products/Categories เป็น public routes (ไม่ต้องการ auth) → ไม่ต้องส่ง token
 *
 * CQRS: ฝั่งลูกค้าอ่านสินค้าจาก catalog_service (read model, MongoDB) ผ่าน /api/catalog/*
 *       ส่วน admin ยังใช้ product_service (write model, PostgreSQL) ผ่าน /api/products/*
 */

import type { Product, ProductListResponse, Category, Variant, ProductCategory } from "./types";

const BFF = process.env.BFF_URL ?? "http://localhost:8080";

/* ─── Catalog read model shapes ───
 * catalog_service (MongoDB) ส่ง field คนละชื่อกับ product_service:
 *   product_id (ไม่ใช่ id), variant_id, attributes[{key,value}] (ไม่ใช่ options[{name,value}])
 * เรา map กลับเป็น Product/Variant ของ frontend เพื่อให้หน้า shop ใช้ type เดิมได้
 */
interface CatalogVariantRaw {
  variant_id: number;
  sku: string;
  name: string;
  price: number;
  stock: number;
  is_active: boolean;
  image_urls: string[] | null;
  attributes: { key: string; value: string }[] | null;
}

interface CatalogCategoryRaw {
  category_id: number;
  name: string;
  slug: string;
}

interface CatalogProductRaw {
  product_id: number;
  name: string;
  description: string;
  image_urls: string[] | null;
  categories: CatalogCategoryRaw[] | null;
  variants: CatalogVariantRaw[] | null;
  is_active: boolean;
}

interface CatalogListRaw {
  items: CatalogProductRaw[] | null;
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

function mapCatalogVariant(v: CatalogVariantRaw): Variant {
  return {
    id: v.variant_id,
    sku: v.sku,
    name: v.name,
    price: v.price,
    stock: v.stock,
    is_active: v.is_active,
    image_urls: v.image_urls ?? [],
    options: (v.attributes ?? []).map((a) => ({ name: a.key, value: a.value })),
  };
}

function mapCatalogCategory(c: CatalogCategoryRaw): ProductCategory {
  return { id: c.category_id, name: c.name };
}

function mapCatalogProduct(c: CatalogProductRaw): Product {
  return {
    id: c.product_id,
    name: c.name,
    description: c.description,
    image_urls: c.image_urls ?? [],
    is_active: c.is_active,
    variants: (c.variants ?? []).map(mapCatalogVariant),
    categories: (c.categories ?? []).map(mapCatalogCategory),
    created_by: 0,
  };
}

// ลูกค้าอ่านสินค้าจาก catalog_service (CQRS read model) — admin ใช้ product_service ตามเดิม
export async function serverFetchProducts(params: {
  page?: number;
  limit?: number;
  search?: string;
  category?: string;
}): Promise<ProductListResponse> {
  const q = new URLSearchParams();
  if (params.page)     q.set("page",        String(params.page));
  if (params.limit)    q.set("limit",       String(params.limit));
  if (params.search)   q.set("search",      params.search);
  if (params.category) q.set("category_id", params.category);

  try {
    const res = await fetch(`${BFF}/api/catalog/products?${q}`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) return { items: [], total: 0, page: 1, page_size: 20, total_pages: 0 };
    const data: CatalogListRaw = await res.json();
    return {
      items: (data.items ?? []).map(mapCatalogProduct),
      total: data.total ?? 0,
      page: data.page ?? 1,
      page_size: data.page_size ?? 20,
      total_pages: data.total_pages ?? 0,
    };
  } catch {
    return { items: [], total: 0, page: 1, page_size: 20, total_pages: 0 };
  }
}

export async function serverFetchProduct(id: number): Promise<Product | null> {
  try {
    const res = await fetch(`${BFF}/api/catalog/products/${id}`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) return null;
    const data: CatalogProductRaw = await res.json();
    return mapCatalogProduct(data);
  } catch {
    return null;
  }
}

export async function serverFetchCategories(): Promise<Category[]> {
  try {
    const res = await fetch(`${BFF}/api/categories`, {
      next: { revalidate: 300 },
    });
    if (!res.ok) return [];
    // backend returns Category[] directly (not wrapped)
    const data = await res.json();
    return Array.isArray(data) ? data : [];
  } catch {
    return [];
  }
}
