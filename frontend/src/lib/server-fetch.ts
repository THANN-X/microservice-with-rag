/**
 * server-fetch.ts — Server-side fetch utilities สำหรับ public API
 *
 * Why: api.ts ใช้ browser-specific features (in-memory accessToken, credentials: "include")
 *      Server Components ไม่มี browser context → ใช้ api.ts ไม่ได้
 *      Next.js rewrites ใช้ได้เฉพาะ browser request เท่านั้น → SC ต้อง fetch ตรงไป BFF
 *      Products/Categories เป็น public routes (ไม่ต้องการ auth) → ไม่ต้องส่ง token
 */

import type { Product, ProductListResponse, Category } from "./types";

const BFF = process.env.BFF_URL ?? "http://localhost:8080";

export async function serverFetchProducts(params: {
  page?: number;
  limit?: number;
  search?: string;
  category?: string;
}): Promise<ProductListResponse> {
  const q = new URLSearchParams();
  if (params.page)     q.set("page",      String(params.page));
  if (params.limit)    q.set("limit",     String(params.limit));
  if (params.search)   q.set("search",    params.search);
  if (params.category) q.set("category",  params.category);
  q.set("is_active", "true");

  try {
    const res = await fetch(`${BFF}/api/products?${q}`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) return { items: [], total: 0, page: 1, page_size: 20, total_pages: 0 };
    return await res.json();
  } catch {
    return { items: [], total: 0, page: 1, page_size: 20, total_pages: 0 };
  }
}

export async function serverFetchProduct(id: number): Promise<Product | null> {
  try {
    const res = await fetch(`${BFF}/api/products/${id}`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) return null;
    return await res.json();
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
