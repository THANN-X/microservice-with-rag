// WHAT: แท็บ "ข้อมูลทั่วไป" ของ modal แก้ไขสินค้า — ชื่อ / รายละเอียด / หมวดหมู่ / สถานะ
"use client";

import { ToggleTrack } from "@/components/admin/toggle-track";
import { fieldLabel, inputField } from "@/components/admin/form-styles";
import { cn } from "@/lib/utils";

export function EditorGeneralTab({
  name,
  onNameChange,
  description,
  onDescriptionChange,
  categoryId,
  onCategoryChange,
  categories,
  isActive,
  onActiveChange,
}: {
  name: string;
  onNameChange: (v: string) => void;
  description: string;
  onDescriptionChange: (v: string) => void;
  categoryId: number | "";
  onCategoryChange: (v: number | "") => void;
  categories: { id: number; label: string }[];
  isActive: boolean;
  onActiveChange: (v: boolean) => void;
}) {
  return (
    <div className="space-y-4">
      <div>
        <label className={fieldLabel}>ชื่อสินค้า *</label>
        <input
          type="text"
          value={name}
          maxLength={255}
          onChange={(e) => onNameChange(e.target.value)}
          className={inputField}
        />
      </div>

      <div>
        <div className="mb-1 flex items-center justify-between">
          <label className="text-xs font-medium text-secondary">รายละเอียด *</label>
          <span
            className={cn("text-[10px]", description.length > 255 ? "text-error" : "text-outline")}
          >
            {description.length}/255
          </span>
        </div>
        <textarea
          value={description}
          onChange={(e) => onDescriptionChange(e.target.value)}
          rows={4}
          className={cn(
            inputField,
            "resize-none",
            description.length > 255 && "ring-2 ring-error/40"
          )}
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={fieldLabel}>หมวดหมู่ *</label>
          <select
            value={categoryId}
            onChange={(e) => onCategoryChange(e.target.value ? Number(e.target.value) : "")}
            className={inputField}
          >
            <option value="">-- เลือกหมวดหมู่ --</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className={fieldLabel}>สถานะสินค้า</label>
          {/* toggle switch จริง — เดิมเป็น badge ที่กดได้แต่ไม่มีใครรู้ว่ากดได้ */}
          <button
            role="switch"
            aria-checked={isActive}
            onClick={() => onActiveChange(!isActive)}
            className="flex w-full items-center gap-3 rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm transition-colors hover:bg-surface-low/70"
          >
            <ToggleTrack on={isActive} />
            <span className={cn("font-medium", isActive ? "text-emerald-700" : "text-secondary")}>
              {isActive ? "แสดงบนหน้าร้าน" : "ซ่อนจากหน้าร้าน"}
            </span>
          </button>
        </div>
      </div>
    </div>
  );
}
