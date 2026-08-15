// WHAT: กรอบ modal ที่ใช้ร่วมกันทุกหน้า admin
// WHY:  พฤติกรรมพื้นฐานที่ผู้ใช้คาดหวัง (Esc ปิด / คลิกฉากหลังปิด / ล็อค scroll พื้นหลัง)
//       ถ้าปล่อยให้แต่ละหน้าเขียน modal เอง จะหลุดข้อใดข้อหนึ่งเสมอ — รวมไว้ที่เดียว
// NOTE: header/footer อยู่นอกพื้นที่ scroll → ปุ่มบันทึกกับข้อความ feedback
//       จะมองเห็นตลอดแม้เนื้อหาจะยาว
"use client";

import { useEffect, type ReactNode } from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

export function ModalShell({
  open,
  onClose,
  title,
  subtitle,
  size = "lg",
  tabs,
  footer,
  children,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle?: string;
  size?: "lg" | "2xl" | "3xl";
  /** แถบแท็บ — วางใต้ header และอยู่นอกพื้นที่ scroll เพื่อให้สลับแท็บได้ตลอด */
  tabs?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
}) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    // กันหน้าเว็บด้านหลังเลื่อนตามขณะ modal เปิด
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = prevOverflow;
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={title}
      onClick={onClose}
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/30 p-4 backdrop-blur-sm"
    >
      {/* stopPropagation: คลิกในตัว modal ต้องไม่ทะลุไปโดน onClose ของฉากหลัง */}
      <div
        onClick={(e) => e.stopPropagation()}
        className={cn(
          "relative flex max-h-[90vh] w-full flex-col rounded-2xl bg-white shadow-xl animate-in fade-in zoom-in-95 duration-200",
          size === "3xl" ? "max-w-3xl" : size === "2xl" ? "max-w-2xl" : "max-w-lg"
        )}
      >
        <div
          className={cn(
            "flex items-start justify-between gap-4 px-8 py-5",
            !tabs && "border-b border-surface-highest"
          )}
        >
          <div className="min-w-0">
            <h2 className="text-lg font-bold text-on-surface">{title}</h2>
            {subtitle && <p className="truncate text-xs text-secondary">{subtitle}</p>}
          </div>
          <button
            onClick={onClose}
            aria-label="ปิด"
            className="shrink-0 rounded-full p-1.5 text-secondary hover:bg-surface-highest hover:text-on-surface"
          >
            <X size={20} />
          </button>
        </div>

        {tabs}

        <div className="flex-1 overflow-y-auto px-8 py-6">{children}</div>

        {footer && <div className="border-t border-surface-highest px-8 py-4">{footer}</div>}
      </div>
    </div>
  );
}
