// WHAT: แถบข้อความ error / success แบบ inline ใช้ในฟอร์มและ banner ของหน้า admin
import { AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

export function FormMessage({ ok, text }: { ok: boolean; text: string }) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-xl px-4 py-2.5 text-xs font-medium",
        ok ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-600"
      )}
    >
      {!ok && <AlertCircle size={14} className="shrink-0" />}
      <span>{text}</span>
    </div>
  );
}
