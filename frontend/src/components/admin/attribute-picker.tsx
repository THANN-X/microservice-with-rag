// WHAT: เลือก attribute value ของ variant แบบ chip กดสลับ (เช่น สี=แดง, ไซส์=M)
// WHY:  ก้อนนี้เคยถูก copy ไว้ 2 จุดในหน้า products (ฟอร์มสร้างสินค้า / ฟอร์มเพิ่ม variant)
//       แก้ตรงหนึ่งแล้วลืมอีกตรงหนึ่งทุกครั้ง
"use client";

import type { Attribute } from "@/lib/types";
import { cn } from "@/lib/utils";

export function AttributePicker({
  attributes,
  selected,
  onChange,
  emptyText = "ยังไม่มี attributes — สร้างได้ที่หน้า Attributes",
}: {
  attributes: Attribute[];
  selected: number[];
  onChange: (ids: number[]) => void;
  emptyText?: string;
}) {
  if (attributes.length === 0) {
    return <p className="text-xs text-outline">{emptyText}</p>;
  }

  const toggle = (id: number) =>
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]);

  return (
    <div className="space-y-2 rounded-xl bg-surface-low/40 p-3">
      {attributes.map((attr) => (
        <div key={attr.id}>
          <p className="mb-1 text-xs font-semibold text-on-surface">{attr.name}</p>
          <div className="flex flex-wrap gap-2">
            {(attr.values ?? []).map((val) => {
              const checked = selected.includes(val.id);
              return (
                <button
                  key={val.id}
                  type="button"
                  onClick={() => toggle(val.id)}
                  className={cn(
                    "rounded-lg border px-3 py-1 text-xs font-medium transition-colors",
                    checked
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-surface-highest bg-white text-secondary hover:border-primary/40"
                  )}
                >
                  {val.value}
                </button>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}
