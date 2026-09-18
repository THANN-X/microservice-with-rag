// WHAT: สร้างสินค้าใหม่ + variant แรกใน 1 form
"use client";

import { useCallback, useState } from "react";
import { ModalShell } from "@/components/admin/modal-shell";
import { FormMessage } from "@/components/admin/form-message";
import { CloudinaryImageUploader } from "@/components/admin/cloudinary-uploader";
import { AttributePicker } from "@/components/admin/attribute-picker";
import { ConfirmDialog } from "@/components/admin/confirm-dialog";
import { fieldLabel, inputField } from "@/components/admin/form-styles";
import { adminProductService } from "@/lib/services";
import { toUserMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";
import type { Attribute, CreateProductRequest } from "@/lib/types";

const EMPTY_PRODUCT_FORM = {
  name: "",
  description: "",
  image_urls: [] as string[],
  category_ids: [] as number[],
  variantName: "",
  sku: "",
  price: "",
  stock: "",
  attribute_value_ids: [] as number[],
};

export function AddProductModal({
  open,
  onClose,
  categories,
  attributes,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  categories: { id: number; label: string }[];
  attributes: Attribute[];
  onCreated: (message: string) => void;
}) {
  const [form, setForm] = useState(EMPTY_PRODUCT_FORM);
  const [saving, setSaving] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  // ฟอร์มยังว่างอยู่ไหม — ใช้ตัดสินว่าจะเตือนก่อนปิดหรือไม่
  const isDirty =
    form.name !== "" || form.description !== "" || form.sku !== "" ||
    form.variantName !== "" || form.price !== "" || form.stock !== "" ||
    form.image_urls.length > 0 || form.category_ids.length > 0 ||
    form.attribute_value_ids.length > 0;

  // WHAT: รายชื่อ field ที่ยังไม่ได้กรอก
  // WHY:  เดิมฟอร์มติดดาว * แค่ 3 ช่อง แต่ backend (CreateProductReq/CreateVariantReq)
  //       บังคับมากกว่านั้นมาก — รายละเอียด, รูปอย่างน้อย 1, หมวดหมู่อย่างน้อย 1,
  //       ราคา > 0, สต็อก > 0, attribute อย่างน้อย 1
  //       ผู้ใช้ที่กรอกแค่ช่องติดดาวจึงโดน 400 ทุกครั้งโดยไม่รู้สาเหตุ
  const missingFields = [
    !form.name.trim() && "ชื่อสินค้า",
    !form.description.trim() && "รายละเอียด",
    form.description.length > 255 && "รายละเอียดเกิน 255 ตัวอักษร",
    form.image_urls.length === 0 && "รูปภาพอย่างน้อย 1 รูป",
    form.category_ids.length === 0 && "หมวดหมู่",
    !form.variantName.trim() && "ชื่อ Variant",
    !form.sku.trim() && "SKU",
    !(parseFloat(form.price) > 0) && "ราคา (>0)",
    !(parseInt(form.stock) > 0) && "สต็อก (>0)",
    form.attribute_value_ids.length === 0 && "Attribute อย่างน้อย 1",
  ].filter(Boolean) as string[];

  const discardAndClose = useCallback(() => {
    setForm(EMPTY_PRODUCT_FORM);
    setErrorMsg("");
    setShowCloseConfirm(false);
    onClose();
  }, [onClose]);

  // WHY: กด Esc / คลิกฉากหลังก็ปิด modal ได้ → เสี่ยงปิดโดนโดยไม่ตั้งใจ
  //      ถ้ามีข้อมูลกรอกค้างอยู่ต้องถามก่อน ไม่งั้นงานที่พิมพ์มาหายทั้งหมด
  const requestClose = useCallback(() => {
    if (isDirty) {
      setShowCloseConfirm(true);
      return;
    }
    discardAndClose();
  }, [isDirty, discardAndClose]);

  const handleSubmit = async () => {
    const { name, description, image_urls, category_ids, variantName, sku, price, stock, attribute_value_ids } = form;

    if (!name.trim() || !sku.trim() || !variantName.trim()) return;
    setSaving(true);
    setErrorMsg("");
    try {
      const req: CreateProductRequest = {
        name: name.trim(),
        description,
        image_urls,
        category_ids,
        variants: [
          {
            sku: sku.trim(),
            name: variantName.trim(),
            price: parseFloat(price) || 0,
            stock: parseInt(stock) || 0,
            attribute_value_ids,
          },
        ],
      };
      await adminProductService.create(req);
      setForm(EMPTY_PRODUCT_FORM);
      onCreated(`เพิ่มสินค้า "${name.trim()}" เรียบร้อย`);
      onClose();
    } catch (err) {
      // WHY: เดิมไม่มี catch เลย — API พัง (เช่น SKU ซ้ำ) แล้ว modal ค้างเงียบๆ
      //      ผู้ใช้ไม่รู้ว่าเกิดอะไรขึ้นแล้วกดซ้ำ
      setErrorMsg(
        toUserMessage(err, "บันทึกสินค้าไม่สำเร็จ — ตรวจสอบว่า SKU ไม่ซ้ำกับสินค้าเดิม แล้วลองใหม่อีกครั้ง")
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <ModalShell
        open={open}
        onClose={requestClose}
        title="เพิ่มสินค้าใหม่"
        subtitle="ช่องที่มี * จำเป็นต้องกรอก"
        footer={
          <div className="space-y-3">
            {errorMsg && <FormMessage ok={false} text={errorMsg} />}
            {missingFields.length > 0 && (
              <p className="text-xs text-outline">
                ยังต้องกรอก:{" "}
                <span className="font-medium text-secondary">{missingFields.join(", ")}</span>
              </p>
            )}
            <div className="flex justify-end gap-3">
              <button
                onClick={requestClose}
                className="rounded-xl px-5 py-2.5 text-sm font-medium text-secondary hover:bg-surface-highest"
              >
                ยกเลิก
              </button>
              <button
                onClick={handleSubmit}
                disabled={saving || missingFields.length > 0}
                className="gradient-primary rounded-xl px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl disabled:opacity-50"
              >
                {saving ? "กำลังบันทึก..." : "บันทึกสินค้า"}
              </button>
            </div>
          </div>
        }
      >
        <div className="space-y-4">
          {/* Image */}
          <div>
            <label className={fieldLabel}>
              รูปภาพสินค้า * <span className="font-normal text-outline">(อย่างน้อย 1 รูป)</span>
            </label>
            <CloudinaryImageUploader
              urls={form.image_urls}
              onChange={(urls) => setForm((f) => ({ ...f, image_urls: urls }))}
            />
          </div>

          {/* Name */}
          <div>
            <label className={fieldLabel}>ชื่อสินค้า *</label>
            <input
              type="text"
              placeholder="ชื่อสินค้า"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              className={inputField}
            />
          </div>

          {/* Category */}
          <div>
            <label className={fieldLabel}>หมวดหมู่ *</label>
            <select
              value={form.category_ids[0] ?? ""}
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  category_ids: e.target.value ? [Number(e.target.value)] : [],
                }))
              }
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

          {/* Description */}
          <div>
            <div className="mb-1 flex items-center justify-between">
              <label className="text-xs font-medium text-secondary">รายละเอียด *</label>
              <span
                className={cn(
                  "text-[10px]",
                  form.description.length > 255 ? "text-error" : "text-outline"
                )}
              >
                {form.description.length}/255
              </span>
            </div>
            <textarea
              placeholder="รายละเอียดสินค้า..."
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
              rows={3}
              className={cn(
                inputField,
                "resize-none",
                form.description.length > 255 && "ring-2 ring-error/40"
              )}
            />
          </div>

          {/* Variant section separator */}
          <div className="flex items-center gap-3 pt-1">
            <div className="h-px flex-1 bg-surface-highest" />
            <span className="text-xs font-semibold text-secondary">Variant แรก</span>
            <div className="h-px flex-1 bg-surface-highest" />
          </div>

          {/* Variant Name */}
          <div>
            <label className={fieldLabel}>
              ชื่อ Variant *
              <span className="ml-1 font-normal text-outline">(เช่น สีดำ / ไซส์ M / Standard)</span>
            </label>
            <input
              type="text"
              placeholder="เช่น สีดำ ไซส์ M"
              value={form.variantName}
              onChange={(e) => setForm((f) => ({ ...f, variantName: e.target.value }))}
              className={inputField}
            />
          </div>

          {/* Variant row */}
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className={fieldLabel}>SKU *</label>
              <input
                type="text"
                placeholder="SKU-001"
                value={form.sku}
                onChange={(e) => setForm((f) => ({ ...f, sku: e.target.value }))}
                className={inputField}
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
                className={inputField}
              />
            </div>
            <div>
              <label className={fieldLabel}>สต็อก *</label>
              <input
                type="number"
                min={1}
                step={1}
                placeholder="ต้องมากกว่า 0"
                value={form.stock}
                onChange={(e) => setForm((f) => ({ ...f, stock: e.target.value }))}
                className={inputField}
              />
            </div>
          </div>

          {/* Attribute Values */}
          <div>
            <label className="mb-2 block text-xs font-medium text-secondary">
              Attributes (Variant แรก) *{" "}
              <span className="font-normal text-outline">(เลือกอย่างน้อย 1)</span>
            </label>
            <AttributePicker
              attributes={attributes}
              selected={form.attribute_value_ids}
              onChange={(ids) => setForm((f) => ({ ...f, attribute_value_ids: ids }))}
            />
          </div>
        </div>
      </ModalShell>

      <ConfirmDialog
        open={showCloseConfirm}
        title="ปิดโดยไม่บันทึก?"
        message="มีข้อมูลที่กรอกไว้แล้วแต่ยังไม่ได้บันทึก ถ้าปิดตอนนี้ข้อมูลทั้งหมดจะหายไป"
        confirmLabel="ปิดโดยไม่บันทึก"
        cancelLabel="กลับไปกรอกต่อ"
        danger
        onConfirm={discardAndClose}
        onCancel={() => setShowCloseConfirm(false)}
      />
    </>
  );
}
