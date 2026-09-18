// WHAT: ฟอร์มเพิ่ม variant ใหม่ ในแท็บ Variants ของ modal แก้ไขสินค้า
// WHY:  เป็นการ "สร้างใหม่" ไม่ใช่การแก้ไข draft จึงมีปุ่มบันทึกของตัวเอง
//       แยกจากปุ่ม "บันทึกการเปลี่ยนแปลง" รวมของ modal
"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { AttributePicker } from "@/components/admin/attribute-picker";
import { fieldLabel, inputFieldSm } from "@/components/admin/form-styles";
import { adminProductService } from "@/lib/services";
import { toUserMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import type { AddVariantRequest, Attribute } from "@/lib/types";

const EMPTY_VARIANT_FORM = {
  sku: "",
  name: "",
  price: "",
  stock: "",
  attribute_value_ids: [] as number[],
};

export function AddVariantForm({
  productId,
  attributes,
  onAdded,
  onError,
}: {
  productId: number;
  attributes: Attribute[];
  /** เพิ่มสำเร็จ — ฝั่ง page จะ refetch, ปิด modal แล้วโชว์ banner */
  onAdded: (message: string) => void;
  onError: (message: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const [form, setForm] = useState(EMPTY_VARIANT_FORM);
  const [saving, setSaving] = useState(false);

  // backend บังคับ price>0, stock>0, attribute อย่างน้อย 1 (ดู AddVariantReq)
  const missing = [
    !form.sku.trim() && "SKU",
    !(parseFloat(form.price) > 0) && "ราคา (>0)",
    !(parseInt(form.stock) > 0) && "สต็อก (>0)",
    form.attribute_value_ids.length === 0 && "Attribute อย่างน้อย 1",
  ].filter(Boolean) as string[];

  const handleAdd = async () => {
    if (!form.sku.trim()) return;
    const label = form.name.trim() || form.sku.trim();
    setSaving(true);
    try {
      await adminProductService.addVariant(productId, {
        sku: form.sku.trim(),
        name: label,
        price: parseFloat(form.price) || 0,
        stock: parseInt(form.stock) || 0,
        attribute_value_ids: form.attribute_value_ids,
      } as AddVariantRequest);
      setForm(EMPTY_VARIANT_FORM);
      setExpanded(false);
      onAdded(`เพิ่ม variant "${label}" เรียบร้อย`);
    } catch (err) {
      onError(toUserMessage(err, "เพิ่ม variant ไม่สำเร็จ — ตรวจสอบว่า SKU ไม่ซ้ำ"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="rounded-xl border border-dashed border-surface-highest">
      <button
        onClick={() => setExpanded((v) => !v)}
        className="flex w-full items-center gap-2 px-5 py-3 text-sm font-semibold text-secondary hover:text-primary"
      >
        <Plus size={16} className={cn("transition-transform", expanded && "rotate-45")} />
        เพิ่ม Variant ใหม่
      </button>
      {expanded && (
        <div className="border-t border-surface-highest p-5">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className={fieldLabel}>SKU *</label>
              <input
                type="text"
                placeholder="SKU-001"
                value={form.sku}
                onChange={(e) => setForm((f) => ({ ...f, sku: e.target.value }))}
                className={inputFieldSm}
              />
            </div>
            <div>
              <label className={fieldLabel}>ชื่อ variant</label>
              <input
                type="text"
                placeholder="เช่น สีแดง / ไซส์ M"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                className={inputFieldSm}
              />
            </div>
            <div>
              <label className={fieldLabel}>ราคา (฿) *</label>
              <input
                type="number"
                min={0.01}
                step="0.01"
                placeholder="ต้องมากกว่า 0"
                value={form.price}
                onChange={(e) => setForm((f) => ({ ...f, price: e.target.value }))}
                className={inputFieldSm}
              />
            </div>
            <div>
              <label className={fieldLabel}>สต็อกเริ่มต้น *</label>
              <input
                type="number"
                min={1}
                step={1}
                placeholder="ต้องมากกว่า 0"
                value={form.stock}
                onChange={(e) => setForm((f) => ({ ...f, stock: e.target.value }))}
                className={inputFieldSm}
              />
            </div>
            <div className="col-span-2">
              <label className={fieldLabel}>Attribute Values *</label>
              <AttributePicker
                attributes={attributes}
                selected={form.attribute_value_ids}
                onChange={(ids) => setForm((f) => ({ ...f, attribute_value_ids: ids }))}
              />
            </div>
          </div>
          <div className="mt-4 flex items-center justify-end gap-3">
            {missing.length > 0 && (
              <p className="text-[11px] text-outline">ยังต้องกรอก: {missing.join(", ")}</p>
            )}
            <button
              onClick={handleAdd}
              disabled={saving || missing.length > 0}
              className="gradient-primary rounded-xl px-5 py-2 text-xs font-semibold text-white shadow-lg shadow-primary/25 disabled:opacity-50"
            >
              {saving ? "กำลังเพิ่ม..." : "เพิ่ม Variant"}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
