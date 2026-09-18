// WHAT: แถบแท็บพร้อม badge นับการแก้ไขที่ยังไม่บันทึกในแท็บนั้น
// WHY:  ผู้ใช้แก้หลายแท็บแล้วจำไม่ได้ว่าค้างตรงไหน — ตัวเลขบนแท็บบอกได้ทันที
"use client";

import { cn } from "@/lib/utils";

export function TabBar<T extends string>({
  tabs,
  active,
  onChange,
}: {
  tabs: { key: T; label: string; badge?: number }[];
  active: T;
  onChange: (key: T) => void;
}) {
  return (
    <div className="flex gap-1 border-b border-surface-highest px-8">
      {tabs.map((t) => (
        <button
          key={t.key}
          onClick={() => onChange(t.key)}
          className={cn(
            "relative flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors",
            active === t.key
              ? "border-primary text-primary"
              : "border-transparent text-secondary hover:text-on-surface"
          )}
        >
          {t.label}
          {t.badge ? (
            <span className="rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-bold text-amber-700">
              {t.badge}
            </span>
          ) : null}
        </button>
      ))}
    </div>
  );
}
