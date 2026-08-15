/**
 * use-product-list.ts — state ของตารางสินค้าในหน้า admin
 *
 * What: โหลดสินค้าตาม page/search/filter + โหลด master data (หมวดหมู่, attribute) ตอน mount
 * Why:  แยกออกจาก page.tsx เพื่อให้ตัวหน้าเหลือแต่ layout — ตรรกะ debounce / reset page /
 *       แปลง category tree เป็น list แบน อยู่รวมกันที่เดียว
 */
"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { attributeService, categoryService, productService } from "@/lib/services";
import { APP_CONFIG } from "@/lib/constants";
import type { Attribute, Category, Product } from "@/lib/types";

export type StatusFilter = "" | "active" | "inactive";

/** เลขหน้าที่จะโชว์เป็นปุ่ม — เลื่อนหน้าต่างตามหน้าปัจจุบัน สูงสุด MAX_BUTTONS ปุ่ม */
function paginationRange(page: number, totalPages: number, maxButtons = 5): number[] {
  let start = Math.max(1, page - 2);
  const end = Math.min(totalPages, start + maxButtons - 1);
  // ใกล้หน้าสุดท้ายแล้วหน้าต่างจะสั้นลง — ดึง start ถอยหลังให้ครบจำนวนปุ่ม
  if (end - start + 1 < maxButtons) start = Math.max(1, end - maxButtons + 1);
  return Array.from({ length: end - start + 1 }, (_, i) => start + i);
}

export function useProductList() {
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [attributes, setAttributes] = useState<Attribute[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [categoryFilter, setCategoryFilter] = useState<number | "">("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("");
  const [fetchError, setFetchError] = useState(false);
  const limit = APP_CONFIG.PAGINATION.ADMIN_TABLE;

  // โหลด master data ครั้งเดียวตอน mount — เข้า/ออก route นี้ใหม่จะโหลดซ้ำ
  useEffect(() => {
    categoryService.list().then(setCategories).catch(() => {});
    attributeService.list().then(setAttributes).catch(() => {});
  }, []);

  // useCallback: ป้องกัน function re-create ทุก render → dependency ใน useEffect ไม่กระตุกซ้ำ
  const fetchProducts = useCallback(async () => {
    setLoading(true);
    setFetchError(false);
    try {
      const res = await productService.list({
        page,
        limit,
        search: debouncedSearch || undefined,
        category: categoryFilter === "" ? undefined : categoryFilter,
        // undefined = ดึงทั้งหมด, true/false = กรองตามสถานะ
        is_active: statusFilter === "" ? undefined : statusFilter === "active",
      });
      setProducts(res.items ?? []);
      setTotal(res.total ?? 0);
    } catch {
      setFetchError(true);
      setProducts([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, limit, debouncedSearch, categoryFilter, statusFilter]);

  useEffect(() => {
    fetchProducts();
  }, [fetchProducts]);

  // Debounce: หน่วง 300ms ก่อนยิง API
  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(t);
  }, [search]);

  // Reset กลับหน้า 1 เมื่อเงื่อนไขค้นหา/กรองเปลี่ยน
  // (ไม่งั้นค้างอยู่หน้า 5 ของผลลัพธ์เดิมที่อาจมีแค่ 1 หน้า → เจอตารางว่าง)
  useEffect(() => {
    // เช็คก่อนว่าถ้าไม่ได้อยู่หน้า 1 ค่อยเซ็ต จะได้ไม่ทริกเกอร์ให้มันเรนเดอร์ซ้ำซ้อนฟรีๆ
    if (page !== 1) setPage(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedSearch, categoryFilter, statusFilter]);

  // WHAT: แปลง category tree เป็น list แบน ๆ เพื่อใช้ใน dropdown
  // WHY:  categories จาก API เป็น tree (มี children) — เดิม dropdown แสดงแค่ระดับบนสุด
  //       ทำให้เลือกหมวดหมู่ย่อยไม่ได้เลย
  const flatCategories = useMemo(() => {
    const out: { id: number; label: string }[] = [];
    const walk = (nodes: Category[], depth: number) => {
      nodes.forEach((c) => {
        out.push({ id: c.id, label: `${"  ".repeat(depth)}${depth > 0 ? "└ " : ""}${c.name}` });
        if (c.children?.length) walk(c.children, depth + 1);
      });
    };
    walk(categories, 0);
    return out;
  }, [categories]);

  const hasActiveFilter =
    debouncedSearch !== "" || categoryFilter !== "" || statusFilter !== "";

  const clearFilters = useCallback(() => {
    setSearch("");
    setCategoryFilter("");
    setStatusFilter("");
  }, []);

  const totalPages = Math.ceil(total / limit) || 1;

  return {
    // ข้อมูล
    products,
    attributes,
    flatCategories,
    loading,
    fetchError,
    refetch: fetchProducts,
    // ตัวกรอง
    search, setSearch,
    categoryFilter, setCategoryFilter,
    statusFilter, setStatusFilter,
    hasActiveFilter,
    clearFilters,
    // การแบ่งหน้า
    page, setPage,
    total,
    limit,
    totalPages,
    pageButtons: paginationRange(page, totalPages),
  };
}
