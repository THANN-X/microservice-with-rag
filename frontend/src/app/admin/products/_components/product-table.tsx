// WHAT: ตารางรายการสินค้าในหน้า admin (รวม skeleton ตอนโหลด และสถานะว่าง)
"use client";

import Image from "next/image";
import { Images, Package, Pencil, Trash2 } from "lucide-react";
import { formatBaht, getMinPrice, truncate } from "@/lib/utils";
import type { Product } from "@/lib/types";
import { ActiveBadge, StockBar, VariantOptions } from "./product-badges";

const COLUMNS = ["สินค้า", "SKU", "หมวดหมู่", "ราคา", "สต็อก", "สถานะ"];

const getTotalStock = (p: Product) => p.variants?.reduce((s, v) => s + v.stock, 0) ?? 0;

/** skeleton rows: กันตารางกระตุก/ยุบตัวตอนสลับหน้าหรือเปลี่ยนตัวกรอง */
function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 5 }).map((_, i) => (
        <tr key={`sk-${i}`} className="border-b border-surface-highest/60">
          <td className="px-6 py-4">
            <div className="flex items-center gap-3">
              <div className="h-10 w-10 animate-pulse rounded-lg bg-surface-highest" />
              <div className="h-3 w-40 animate-pulse rounded bg-surface-highest" />
            </div>
          </td>
          {Array.from({ length: 6 }).map((__, j) => (
            <td key={j} className="px-4 py-4">
              <div className="h-3 w-16 animate-pulse rounded bg-surface-highest" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function ProductTable({
  products,
  loading,
  hasActiveFilter,
  onClearFilters,
  onEdit,
  onDelete,
}: {
  products: Product[];
  loading: boolean;
  hasActiveFilter: boolean;
  onClearFilters: () => void;
  /** เปิด modal แก้ไข พร้อมระบุแท็บที่จะเข้าไปเลย */
  onEdit: (product: Product, tab: "general" | "images") => void;
  onDelete: (product: Product) => void;
}) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-surface-highest bg-surface-low/30 text-xs uppercase tracking-wider text-secondary">
            {COLUMNS.map((label, i) => (
              <th key={label} className={i === 0 ? "px-6 py-4 font-medium" : "px-4 py-4 font-medium"}>
                {label}
              </th>
            ))}
            <th className="px-4 py-4 text-right font-medium">จัดการ</th>
          </tr>
        </thead>
        <tbody>
          {loading ? (
            <SkeletonRows />
          ) : products.length === 0 ? (
            <tr>
              <td colSpan={7} className="py-16 text-center text-secondary">
                <Package size={40} className="mx-auto mb-2 text-outline" />
                {hasActiveFilter ? (
                  <>
                    <p>ไม่พบสินค้าที่ตรงกับตัวกรอง</p>
                    <button
                      onClick={onClearFilters}
                      className="mt-2 text-xs font-medium text-primary hover:underline"
                    >
                      ล้างตัวกรองทั้งหมด
                    </button>
                  </>
                ) : (
                  <p>ยังไม่มีสินค้า — กด &quot;เพิ่มสินค้า&quot; เพื่อเริ่มต้น</p>
                )}
              </td>
            </tr>
          ) : (
            products.map((p) => (
              <tr
                key={p.id}
                className="border-b border-surface-highest/60 transition-colors hover:bg-surface-low/40"
              >
                {/* Product */}
                <td className="px-6 py-4">
                  <div className="flex items-center gap-3">
                    {p.image_urls?.[0] ? (
                      <div className="relative h-10 w-10">
                        <Image
                          src={p.image_urls[0]}
                          alt={p.name}
                          fill
                          sizes="40px"
                          className="rounded-lg bg-surface-highest object-cover"
                        />
                      </div>
                    ) : (
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-surface-highest text-outline">
                        <Package size={18} />
                      </div>
                    )}
                    <div className="min-w-0">
                      <span className="font-medium text-on-surface">{truncate(p.name, 35)}</span>
                      {/* บอกจำนวน variant — เดิมสินค้าที่มี 5 variant ดูเหมือนมีตัวเดียว */}
                      {(p.variants?.length ?? 0) > 1 && (
                        <span className="ml-2 rounded-md bg-surface-highest px-1.5 py-0.5 text-[10px] font-medium text-secondary">
                          {p.variants.length} variants
                        </span>
                      )}
                    </div>
                  </div>
                </td>
                {/* SKU */}
                <td className="px-4 py-4">
                  <span className="font-mono text-xs text-secondary">
                    {p.variants?.[0]?.sku ?? "-"}
                  </span>
                  <VariantOptions options={p.variants?.[0]?.options} className="mt-1" />
                </td>
                {/* Category */}
                <td className="px-4 py-4">
                  {p.categories?.[0] ? (
                    <span className="rounded-full bg-primary-container/40 px-2.5 py-0.5 text-[10px] font-semibold text-primary">
                      {p.categories[0].name}
                    </span>
                  ) : (
                    <span className="text-xs text-outline">—</span>
                  )}
                </td>
                {/* Price */}
                <td className="px-4 py-4 font-medium text-on-surface">
                  {formatBaht(getMinPrice(p.variants ?? []))}
                </td>
                {/* Stock */}
                <td className="px-4 py-4">
                  <StockBar stock={getTotalStock(p)} />
                </td>
                {/* Status */}
                <td className="px-4 py-4">
                  <ActiveBadge active={p.is_active} />
                </td>
                {/* Actions */}
                <td className="px-4 py-4 text-right">
                  <div className="flex items-center justify-end gap-1">
                    <button
                      onClick={() => onEdit(p, "images")}
                      className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest hover:text-primary"
                      title="จัดการรูปภาพ"
                    >
                      <Images size={15} />
                    </button>
                    <button
                      onClick={() => onEdit(p, "general")}
                      className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest hover:text-primary"
                      title="แก้ไขสินค้า"
                    >
                      <Pencil size={15} />
                    </button>
                    <button
                      onClick={() => onDelete(p)}
                      className="rounded-lg p-2 text-secondary transition-colors hover:bg-red-50 hover:text-error"
                      title="ลบสินค้า"
                      aria-label={`ลบ ${p.name}`}
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
