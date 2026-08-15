// WHAT: กล่องยืนยันแทน window.confirm()
// WHY:  confirm() ของเบราว์เซอร์บล็อค thread, สไตล์คุมไม่ได้, ขึ้นข้อความยาวไม่สวย
//       และบน modal ซ้อน modal จะดูหลุดจากงานที่ทำอยู่
"use client";

import { useEffect, type ReactNode } from "react";
import { AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "ยืนยัน",
  cancelLabel = "ยกเลิก",
  danger = false,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    // z สูงกว่า ModalShell (60) เพราะต้องซ้อนทับ modal แก้ไขสินค้าได้
    <div
      role="alertdialog"
      aria-modal="true"
      onClick={onCancel}
      className="fixed inset-0 z-[70] flex items-center justify-center bg-black/40 p-4 backdrop-blur-sm"
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl animate-in fade-in zoom-in-95 duration-150"
      >
        <div className="flex gap-3">
          <div
            className={cn(
              "flex h-9 w-9 shrink-0 items-center justify-center rounded-full",
              danger ? "bg-red-50 text-error" : "bg-amber-50 text-amber-600"
            )}
          >
            <AlertCircle size={18} />
          </div>
          <div className="min-w-0">
            <h3 className="text-sm font-bold text-on-surface">{title}</h3>
            <div className="mt-1 text-xs leading-relaxed text-secondary">{message}</div>
          </div>
        </div>
        <div className="mt-5 flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="rounded-xl px-4 py-2 text-xs font-medium text-secondary hover:bg-surface-highest"
          >
            {cancelLabel}
          </button>
          <button
            onClick={onConfirm}
            autoFocus
            className={cn(
              "rounded-xl px-4 py-2 text-xs font-semibold text-white shadow-md transition-all hover:shadow-lg",
              danger ? "bg-error" : "gradient-primary"
            )}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
