// WHAT: แถบแบ่งหน้าใต้ตาราง admin — บอกช่วงที่กำลังแสดง + ปุ่มเลขหน้า
"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

export function PaginationBar({
  page,
  totalPages,
  total,
  limit,
  pageButtons,
  onPageChange,
}: {
  page: number;
  totalPages: number;
  total: number;
  limit: number;
  /** เลขหน้าที่จะแสดงเป็นปุ่ม — คำนวณมาจากฝั่ง hook */
  pageButtons: number[];
  onPageChange: (page: number) => void;
}) {
  return (
    <div className="flex items-center justify-between border-t border-surface-highest px-6 py-4">
      <p className="text-xs text-secondary">
        {total === 0
          ? "ไม่มีรายการ"
          : `แสดง ${(page - 1) * limit + 1}–${Math.min(page * limit, total)} จาก ${total} รายการ`}
      </p>
      <div className="flex items-center gap-1">
        <button
          onClick={() => onPageChange(Math.max(1, page - 1))}
          disabled={page === 1}
          aria-label="หน้าก่อนหน้า"
          className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest disabled:opacity-40"
        >
          <ChevronLeft size={16} />
        </button>
        {pageButtons.map((n) => (
          <button
            key={n}
            onClick={() => onPageChange(n)}
            className={cn(
              "h-8 w-8 rounded-lg text-xs font-medium transition-colors",
              n === page ? "bg-primary text-white" : "text-secondary hover:bg-surface-highest"
            )}
          >
            {n}
          </button>
        ))}
        <button
          onClick={() => onPageChange(Math.min(totalPages, page + 1))}
          disabled={page === totalPages}
          aria-label="หน้าถัดไป"
          className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest disabled:opacity-40"
        >
          <ChevronRight size={16} />
        </button>
      </div>
    </div>
  );
}
