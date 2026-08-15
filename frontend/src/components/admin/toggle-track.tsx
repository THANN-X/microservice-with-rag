// WHAT: ส่วนที่เป็นภาพของ toggle switch (ราง + ปุ่มกลม)
// WHY:  ตัวที่กดจริงคือ <button role="switch"> ของแต่ละที่ ซึ่ง layout ไม่เหมือนกัน
//       แต่ตัวรางกับ animation เหมือนกันเป๊ะ — แยกเฉพาะส่วนภาพมาใช้ร่วม
import { cn } from "@/lib/utils";

export function ToggleTrack({ on, size = "md" }: { on: boolean; size?: "sm" | "md" }) {
  const sm = size === "sm";
  return (
    <span
      className={cn(
        "relative shrink-0 rounded-full transition-colors",
        sm ? "h-4 w-7" : "h-5 w-9",
        on ? "bg-emerald-500" : "bg-surface-highest"
      )}
    >
      <span
        className={cn(
          "absolute top-0.5 rounded-full bg-white shadow transition-all",
          sm ? "h-3 w-3" : "h-4 w-4",
          on ? (sm ? "left-3.5" : "left-[1.125rem]") : "left-0.5"
        )}
      />
    </span>
  );
}
