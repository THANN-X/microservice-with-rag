// WHAT: แถบค้นหา + กรองหมวดหมู่/สถานะ เหนือตารางสินค้า
"use client";

import { Search, X } from "lucide-react";
import { cn } from "@/lib/utils";
import type { StatusFilter } from "../_hooks/use-product-list";

/** select ที่จะเปลี่ยนสีเมื่อมีการกรองอยู่ — ผู้ใช้เห็นได้ทันทีว่าตารางถูกกรองด้วยอะไร */
const selectCls = (active: boolean) =>
  cn(
    "rounded-lg px-3 py-2 text-sm outline-none transition-colors focus:ring-2 focus:ring-primary/20",
    active ? "bg-primary/10 font-medium text-primary" : "bg-surface-highest text-secondary"
  );

export function ProductFilters({
  search,
  onSearchChange,
  categories,
  categoryFilter,
  onCategoryFilterChange,
  statusFilter,
  onStatusFilterChange,
  hasActiveFilter,
  onClearFilters,
}: {
  search: string;
  onSearchChange: (v: string) => void;
  categories: { id: number; label: string }[];
  categoryFilter: number | "";
  onCategoryFilterChange: (v: number | "") => void;
  statusFilter: StatusFilter;
  onStatusFilterChange: (v: StatusFilter) => void;
  hasActiveFilter: boolean;
  onClearFilters: () => void;
}) {
  return (
    <div className="mb-6 flex flex-wrap items-center gap-3 rounded-2xl bg-white p-4 shadow-ambient">
      <div className="relative min-w-[200px] max-w-sm flex-1">
        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-outline" />
        <input
          type="text"
          placeholder="ค้นหาชื่อสินค้า..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          className="w-full rounded-lg bg-surface-highest py-2 pl-9 pr-8 text-sm outline-none transition-all focus:ring-2 focus:ring-primary/20"
        />
        {search && (
          <button
            onClick={() => onSearchChange("")}
            aria-label="ล้างคำค้นหา"
            className="absolute right-2 top-1/2 -translate-y-1/2 rounded-full p-1 text-outline hover:bg-surface-low hover:text-on-surface"
          >
            <X size={12} />
          </button>
        )}
      </div>

      <select
        value={categoryFilter}
        onChange={(e) => onCategoryFilterChange(e.target.value ? Number(e.target.value) : "")}
        className={selectCls(categoryFilter !== "")}
      >
        <option value="">ทุกหมวดหมู่</option>
        {categories.map((c) => (
          <option key={c.id} value={c.id}>
            {c.label}
          </option>
        ))}
      </select>

      <select
        value={statusFilter}
        onChange={(e) => onStatusFilterChange(e.target.value as StatusFilter)}
        className={selectCls(statusFilter !== "")}
      >
        <option value="">ทุกสถานะ</option>
        <option value="active">Active</option>
        <option value="inactive">Inactive</option>
      </select>

      {hasActiveFilter && (
        <button
          onClick={onClearFilters}
          className="flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium text-secondary transition-colors hover:bg-surface-highest hover:text-on-surface"
        >
          <X size={14} />
          ล้างตัวกรอง
        </button>
      )}
    </div>
  );
}
