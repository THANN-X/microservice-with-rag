// WHAT: ชิ้นส่วนแสดงผลเล็ก ๆ ที่ใช้ซ้ำในตารางสินค้าและ modal แก้ไข
import type { VariantOption } from "@/lib/types";
import { cn } from "@/lib/utils";

/* ─── Status badge ─── */
export function ActiveBadge({ active }: { active: boolean }) {
  return (
    <span
      className={cn(
        "rounded-full px-2.5 py-0.5 text-[10px] font-semibold uppercase",
        active ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-600"
      )}
    >
      {active ? "Active" : "Inactive"}
    </span>
  );
}

/* ─── Variant attribute chips ───
 * WHAT: แสดง attribute ของ variant เช่น สี=เทา, ไซส์=M
 * WHY:  API ส่ง variant.options มาให้อยู่แล้ว แต่ UI ไม่เคยเอามาแสดง
 *       เห็นแต่ variant.name ซึ่ง seed ตั้งไว้เป็น "ไซส์ตัวเลือกที่ 1"
 *       → ไม่มีทางรู้เลยว่า variant ไหนคือไซส์อะไร สีอะไร
 */
export function VariantOptions({
  options,
  className,
}: {
  options?: VariantOption[];
  className?: string;
}) {
  if (!options?.length) return null;
  return (
    <div className={cn("flex flex-wrap gap-1", className)}>
      {options.map((o, i) => (
        <span
          key={`${o.name}-${i}`}
          className="inline-flex items-center gap-1 rounded-md bg-surface-highest px-1.5 py-0.5 text-[10px]"
        >
          <span className="text-outline">{o.name}</span>
          <span className="font-semibold text-on-surface">{o.value}</span>
        </span>
      ))}
    </div>
  );
}

/* ─── Stock bar ─── */
export function StockBar({ stock }: { stock: number }) {
  // normalize stock เป็น % โดยถือว่า 200 = เต็ม 100% (capped ไม่เกิน 100%)
  const pct = Math.min(stock / 200, 1) * 100;
  // เกณฑ์สี: แดง = เหลือน้อย (<20), เหลือง = ระวัง (<50), เขียว = โอเค (≥50)
  const color = stock < 20 ? "bg-error" : stock < 50 ? "bg-amber-400" : "bg-primary";
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-surface-highest">
        <div className={cn("h-full rounded-full", color)} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs text-secondary">{stock}</span>
    </div>
  );
}
