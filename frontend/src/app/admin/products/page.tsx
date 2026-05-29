// WHAT: หน้า Admin จัดการสินค้า
// โครงสร้าง component:
//   CloudinaryImageUploader  — อัปโหลดรูปไป Cloudinary แล้วเก็บ URL กลับมา
//   ManageImagesModal        — จัดการรูปของ product + แต่ละ variant แยกกัน
//   AddProductModal          — สร้างสินค้าใหม่ + variant แรกใน 1 form
//   EditProductModal         — แก้ไข info / ราคา / stock / เพิ่ม variant ใหม่
//   AdminProductsPage        — ตารางสินค้า + pagination + search debounce
"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  Plus,
  Search,
  Pencil,
  Trash2,
  X,
  ChevronLeft,
  ChevronRight,
  Package,
  Upload,
  Images,
} from "lucide-react";
import { cn, formatBaht, truncate } from "@/lib/utils";
import {
  productService,
  categoryService,
  adminProductService,
  attributeService,
} from "@/lib/services";
import type {
  Product,
  Category,
  Attribute,
  CreateProductRequest,
  Variant,
  UpdateProductGeneralInfoRequest,
  AdjustStockRequest,
  UpdateVariantPriceRequest,
  SetProductActiveRequest,
  SetVariantActiveRequest,
  AddVariantRequest,
} from "@/lib/types";

/* ─── Status badge ─── */
function ActiveBadge({ active }: { active: boolean }) {
  return (
    <span
      className={cn(
        "rounded-full px-2.5 py-0.5 text-[10px] font-semibold uppercase",
        active
          ? "bg-emerald-50 text-emerald-700"
          : "bg-red-50 text-red-600"
      )}
    >
      {active ? "Active" : "Inactive"}
    </span>
  );
}

/* ─── Stock bar ─── */
function StockBar({ stock }: { stock: number }) {
  // normalize stock เป็น % โดยถือว่า 200 = เต็ม 100% (capped ไม่เกิน 100%)
  const pct = Math.min(stock / 200, 1) * 100;
  // เกณฑ์สี: แดง = เหลือน้อย (<20), เหลือง = ระวัง (<50), เขียว = โอเค (≥50)
  const color =
    stock < 20 ? "bg-error" : stock < 50 ? "bg-amber-400" : "bg-primary";
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-surface-highest">
        <div className={cn("h-full rounded-full", color)} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs text-secondary">{stock}</span>
    </div>
  );
}

/* ─── Cloudinary Image Uploader ─── */
const CLOUDINARY_CLOUD_NAME = process.env.NEXT_PUBLIC_CLOUDINARY_CLOUD_NAME ?? "";
const CLOUDINARY_UPLOAD_PRESET = process.env.NEXT_PUBLIC_CLOUDINARY_UPLOAD_PRESET ?? "";

// WHAT: อัปโหลดไฟล์รูปหลายรูปพร้อมกันไปยัง Cloudinary แล้วเก็บ URL กลับเป็น string[]
// WHY Cloudinary: ไม่ต้อง host รูปเอง — ได้ CDN + auto-resize URL ฟรี
function CloudinaryImageUploader({
  urls,
  onChange,
}: {
  urls: string[];
  onChange: (urls: string[]) => void;
}) {
  const [uploading, setUploading] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");

  const handleFiles = async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    setUploading(true);
    setErrorMsg("");
    try {
      // อัปโหลดทุกไฟล์พร้อมกัน (parallel) — Cloudinary คืน secure_url (HTTPS) ต่อไฟล์
      const uploaded = await Promise.all(
        Array.from(files).map(async (file) => {
          const fd = new FormData();
          fd.append("file", file);
          fd.append("upload_preset", CLOUDINARY_UPLOAD_PRESET);
          const res = await fetch(
            `https://api.cloudinary.com/v1_1/${CLOUDINARY_CLOUD_NAME}/image/upload`,
            { method: "POST", body: fd }
          );
          if (!res.ok) throw new Error("Upload failed");
          const data = (await res.json()) as { secure_url: string };
          return data.secure_url;
        })
      );
      // append URL ใหม่ต่อท้าย URL เดิม (ไม่แทนที่)
      onChange([...urls, ...uploaded]);
    } catch {
      setErrorMsg("อัปโหลดไม่สำเร็จ กรุณาลองใหม่อีกครั้ง");
    } finally {
      setUploading(false);
    }
  };

  const remove = (idx: number) => onChange(urls.filter((_, i) => i !== idx));

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-2">
        {urls.map((url, i) => (
          <div key={i} className="relative">
            <img src={url} alt="" className="h-16 w-16 rounded-lg object-cover bg-surface-highest" />
            <button
              type="button"
              onClick={() => remove(i)}
              className="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-error text-white shadow-sm"
            >
              <X size={10} />
            </button>
          </div>
        ))}
        <label
          className={cn(
            "flex h-16 w-16 cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-surface-highest text-outline transition-colors hover:border-primary/40 hover:text-primary",
            uploading && "cursor-not-allowed opacity-50"
          )}
        >
          {uploading ? (
            <span className="px-1 text-center text-[9px] leading-tight">กำลังอัปโหลด...</span>
          ) : (
            <>
              <Upload size={16} />
              <span className="mt-0.5 text-[9px]">อัปโหลด</span>
            </>
          )}
          <input
            type="file"
            accept="image/*"
            multiple
            disabled={uploading}
            className="hidden"
            onChange={(e) => handleFiles(e.target.files)}
          />
        </label>
      </div>
      {!CLOUDINARY_CLOUD_NAME && (
        <p className="text-[10px] text-amber-600">
          ⚠ กรุณาตั้งค่า NEXT_PUBLIC_CLOUDINARY_CLOUD_NAME และ NEXT_PUBLIC_CLOUDINARY_UPLOAD_PRESET ใน .env.local
        </p>
      )}
      {errorMsg && (
        <p className="text-[10px] text-red-500">{errorMsg}</p>
      )}
    </div>
  );
}

/* ════════  Manage Images Modal  ════════ */
function ManageImagesModal({
  open,
  onClose,
  product,
  onUpdated,
}: {
  open: boolean;
  onClose: () => void;
  product: Product | null;
  onUpdated: () => void;
}) {
  const [productImages, setProductImages] = useState<string[]>([]);
  const [variantImages, setVariantImages] = useState<Record<number, string[]>>({});
  const [saving, setSaving] = useState(false);
  const [savedMsg, setSavedMsg] = useState("");
  // useRef เก็บ timer ID เพื่อ clearTimeout ก่อนตั้งใหม่
  // ป้องกันกรณีกด save เร็วๆ ซ้ำ → timer เก่าจะถูกยกเลิกทุกครั้งก่อน set ใหม่
  const savedMsgTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (product) {
      setProductImages(product.image_urls ?? []);
      const init: Record<number, string[]> = {};
      (product.variants ?? []).forEach((v) => {
        init[v.id] = v.image_urls ?? [];
      });
      setVariantImages(init);
    }
  }, [product]);

  const saveProductImages = async () => {
    if (!product) return;
    setSaving(true);
    try {
      await adminProductService.updateProductImages(product.id, {
        image_urls: productImages,
      });
      setSavedMsg("อัปเดตรูปภาพสินค้าสำเร็จ");
      onUpdated();
    } catch {
      setSavedMsg("เกิดข้อผิดพลาด ลองใหม่อีกครั้ง");
    } finally {
      setSaving(false);
      if (savedMsgTimerRef.current) clearTimeout(savedMsgTimerRef.current);
      savedMsgTimerRef.current = setTimeout(() => setSavedMsg(""), 3000);
    }
  };

  const saveVariantImages = async (variant: Variant) => {
    if (!product) return;
    setSaving(true);
    try {
      await adminProductService.updateVariantImages(product.id, variant.id, {
        image_urls: variantImages[variant.id] ?? [],
      });
      setSavedMsg(`อัปเดตรูปภาพ variant "${variant.name}" สำเร็จ`);
      onUpdated();
    } catch {
      setSavedMsg("เกิดข้อผิดพลาด ลองใหม่อีกครั้ง");
    } finally {
      setSaving(false);
      if (savedMsgTimerRef.current) clearTimeout(savedMsgTimerRef.current);
      savedMsgTimerRef.current = setTimeout(() => setSavedMsg(""), 3000);
    }
  };

  if (!open || !product) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/30 backdrop-blur-sm">
      <div className="relative w-full max-w-lg rounded-2xl bg-white p-8 shadow-xl animate-in fade-in zoom-in-95 duration-200 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-bold text-on-surface">จัดการรูปภาพ</h2>
            <p className="text-xs text-secondary truncate max-w-xs">{product.name}</p>
          </div>
          <button
            onClick={onClose}
            className="rounded-full p-1.5 text-secondary hover:bg-surface-highest hover:text-on-surface"
          >
            <X size={20} />
          </button>
        </div>

        {savedMsg && (
          <div className="mb-4 rounded-xl bg-emerald-50 px-4 py-2.5 text-xs font-medium text-emerald-700">
            {savedMsg}
          </div>
        )}

        <div className="space-y-6">
          {/* Product-level images */}
          <div className="rounded-xl border border-surface-highest p-4">
            <h3 className="mb-3 text-sm font-semibold text-on-surface">รูปภาพหลักของสินค้า</h3>
            <CloudinaryImageUploader urls={productImages} onChange={setProductImages} />
            <div className="mt-3 flex justify-end">
              <button
                onClick={saveProductImages}
                disabled={saving}
                className="gradient-primary rounded-xl px-4 py-2 text-xs font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl disabled:opacity-50"
              >
                บันทึก
              </button>
            </div>
          </div>

          {/* Per-variant images */}
          {(product.variants ?? []).map((v) => (
            <div key={v.id} className="rounded-xl border border-surface-highest p-4">
              <h3 className="mb-1 text-sm font-semibold text-on-surface">{v.name}</h3>
              <p className="mb-3 font-mono text-[10px] text-outline">{v.sku}</p>
              <CloudinaryImageUploader
                urls={variantImages[v.id] ?? []}
                onChange={(urls) => setVariantImages((prev) => ({ ...prev, [v.id]: urls }))}
              />
              <div className="mt-3 flex justify-end">
                <button
                  onClick={() => saveVariantImages(v)}
                  disabled={saving}
                  className="gradient-primary rounded-xl px-4 py-2 text-xs font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl disabled:opacity-50"
                >
                  บันทึก
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/* ════════  Add Product Modal  ════════ */
function AddProductModal({
  open,
  onClose,
  categories,
  attributes,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  categories: Category[];
  attributes: Attribute[];
  onCreated: () => void;
}) {
  const [form, setForm] = useState({
    name: "",
    description: "",
    image_urls: [] as string[],
    category_ids: [] as number[],
    variantName: "",
    sku: "",
    price: "",
    stock: "",
    attribute_value_ids: [] as number[],
  });
  const [saving, setSaving] = useState(false);

  const handleSubmit = async () => {
    const {name, description, image_urls, category_ids, variantName, sku, price, stock, attribute_value_ids} = form;

    if (!name || !sku) return;
    setSaving(true);
    try {
      const req: CreateProductRequest = {
        name,
        description,
        image_urls,
        category_ids,
        variants: [
          {
            sku,
            name: variantName || name,
            price: parseFloat(price) || 0,
            stock: parseInt(stock) || 0,
            attribute_value_ids,
          },
        ],
      };
      await adminProductService.create(req);
      onCreated();
      onClose();
      setForm({
        name: "",
        description: "",
        image_urls: [],
        category_ids: [],
        variantName: "",
        sku: "",
        price: "",
        stock: "",
        attribute_value_ids: [],
      });
    } finally {
      setSaving(false);
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/30 backdrop-blur-sm">
      <div className="relative w-full max-w-lg rounded-2xl bg-white p-8 shadow-xl animate-in fade-in zoom-in-95 duration-200 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-bold text-on-surface">เพิ่มสินค้าใหม่</h2>
            <p className="text-xs text-secondary">กรอกข้อมูลสินค้าด้านล่าง</p>
          </div>
          <button
            onClick={onClose}
            className="rounded-full p-1.5 text-secondary hover:bg-surface-highest hover:text-on-surface"
          >
            <X size={20} />
          </button>
        </div>

        <div className="space-y-4">
          {/* Image */}
          <div>
            <label className="mb-1 block text-xs font-medium text-secondary">
              รูปภาพสินค้า
            </label>
            <CloudinaryImageUploader
              urls={form.image_urls}
              onChange={(urls) => setForm((f) => ({ ...f, image_urls: urls }))}
            />
          </div>

          {/* Name */}
          <div>
            <label className="mb-1 block text-xs font-medium text-secondary">
              ชื่อสินค้า *
            </label>
            <input
              type="text"
              placeholder="ชื่อสินค้า"
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20"
            />
          </div>

          {/* Category */}
          <div>
            <label className="mb-1 block text-xs font-medium text-secondary">
              หมวดหมู่
            </label>
            <select
              value={form.category_ids[0] ?? ""}
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  category_ids: e.target.value ? [Number(e.target.value)] : [],
                }))
              }
              className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20"
            >
              <option value="">-- เลือกหมวดหมู่ --</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          {/* Description */}
          <div>
            <label className="mb-1 block text-xs font-medium text-secondary">
              รายละเอียด
            </label>
            <textarea
              placeholder="รายละเอียดสินค้า..."
              value={form.description}
              onChange={(e) =>
                setForm((f) => ({ ...f, description: e.target.value }))
              }
              rows={3}
              className="w-full resize-none rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20"
            />
          </div>

          {/* Variant row */}
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-secondary">
                SKU *
              </label>
              <input
                type="text"
                placeholder="SKU-001"
                value={form.sku}
                onChange={(e) =>
                  setForm((f) => ({ ...f, sku: e.target.value }))
                }
                className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-secondary">
                ราคา (฿)
              </label>
              <input
                type="number"
                placeholder="0.00"
                value={form.price}
                onChange={(e) =>
                  setForm((f) => ({ ...f, price: e.target.value }))
                }
                className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-secondary">
                สต็อก
              </label>
              <input
                type="number"
                placeholder="0"
                value={form.stock}
                onChange={(e) =>
                  setForm((f) => ({ ...f, stock: e.target.value }))
                }
                className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20"
              />
            </div>
          </div>

          {/* Attribute Values */}
          {attributes.length > 0 && (
            <div>
              <label className="mb-2 block text-xs font-medium text-secondary">
                Attributes (Variant แรก)
              </label>
              <div className="space-y-2 rounded-xl bg-surface-low/40 p-3">
                {attributes.map((attr) => (
                  <div key={attr.id}>
                    <p className="mb-1 text-xs font-semibold text-on-surface">{attr.name}</p>
                    <div className="flex flex-wrap gap-2">
                      {(attr.values ?? []).map((val) => {
                        const checked = form.attribute_value_ids.includes(val.id);
                        return (
                          <button
                            key={val.id}
                            type="button"
                            onClick={() =>
                              setForm((f) => ({
                                ...f,
                                attribute_value_ids: checked
                                  ? f.attribute_value_ids.filter((id) => id !== val.id)
                                  : [...f.attribute_value_ids, val.id],
                              }))
                            }
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
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="mt-6 flex justify-end gap-3">
          <button
            onClick={onClose}
            className="rounded-xl px-5 py-2.5 text-sm font-medium text-secondary hover:bg-surface-highest"
          >
            ยกเลิก
          </button>
          <button
            onClick={handleSubmit}
            disabled={saving || !form.name || !form.sku}
            className="gradient-primary rounded-xl px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl disabled:opacity-50"
          >
            {saving ? "กำลังบันทึก..." : "บันทึกสินค้า"}
          </button>
        </div>
      </div>
    </div>
  );
}

/* ════════  Edit Product Modal  ════════ */
function EditProductModal({
  open,
  onClose,
  product,
  categories,
  attributes,
  onUpdated,
}: {
  open: boolean;
  onClose: () => void;
  product: Product | null;
  categories: Category[];
  attributes: Attribute[];
  onUpdated: () => void;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState<number | "">("");
  const [isActive, setIsActive] = useState(true);
  // variantEdits เก็บ "draft" ของแต่ละ variant ก่อนกด save
  // key = variant.id → ทำให้แก้ได้หลาย variant พร้อมกัน โดยยังไม่ส่ง API
  const [variantEdits, setVariantEdits] = useState<
    Record<number, { price: string; newStock: string; stockReason: string; is_active: boolean }>
  >({});
  const [addForm, setAddForm] = useState({ sku: "", name: "", price: "", stock: "", attribute_value_ids: [] as number[] });
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  // flashTimerRef: เก็บ timer ID สำหรับซ่อน feedback message อัตโนมัติ
  const flashTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!product) return;
    setName(product.name);
    setDescription(product.description ?? "");
    setCategoryId(product.categories?.[0]?.id ?? "");
    setIsActive(product.is_active ?? true);
    const init: typeof variantEdits = {};
    product.variants?.forEach((v) => {
      init[v.id] = {
        price: String(v.price),
        newStock: String(v.stock),
        stockReason: "",
        is_active: v.is_active ?? true,
      };
    });
    setVariantEdits(init);
    setAddForm({ sku: "", name: "", price: "", stock: "", attribute_value_ids: [] });
    setMsg(null);
  }, [product]);

  // flash: แสดง feedback toast (สำเร็จ/ผิดพลาด) แล้วซ่อนอัตโนมัติใน 3 วินาที
  // ถ้าเรียกซ้ำก่อนหมดเวลา → reset timer ใหม่ (ไม่ซ้อนกัน)
  const flash = useCallback((ok: boolean, text: string) => {
    if (flashTimerRef.current) clearTimeout(flashTimerRef.current);
    setMsg({ ok, text });
    flashTimerRef.current = setTimeout(() => setMsg(null), 3000);
  }, []);

  const handleSaveGeneral = async () => {
    if (!product) return;
    setSaving(true);
    try {
      await adminProductService.updateGeneralInfo(product.id, {
        name,
        description,
        category_ids: categoryId !== "" ? [Number(categoryId)] : [],
      } as UpdateProductGeneralInfoRequest);
      if ((product.is_active ?? true) !== isActive) {
        await adminProductService.setProductActive(product.id, { is_active: isActive } as SetProductActiveRequest);
      }
      flash(true, "บันทึกข้อมูลทั่วไปสำเร็จ");
      onUpdated();
    } catch {
      flash(false, "บันทึกไม่สำเร็จ");
    } finally {
      setSaving(false);
    }
  };

  const handleSaveVariant = async (v: Variant) => {
    if (!product) return;
    const edit = variantEdits[v.id];
    if (!edit) return;
    setSaving(true);
    try {
      // รวม API calls เฉพาะ field ที่เปลี่ยนแปลงจริงเท่านั้น
      // price ไม่เปลี่ยน → ไม่ส่ง updateVariantPrice
      // stock เปลี่ยนแต่ไม่มี reason → ไม่ส่ง adjustStock (reason บังคับตาม business rule)
      const ops: Promise<unknown>[] = [];
      if (parseFloat(edit.price) !== v.price) {
        ops.push(
          adminProductService.updateVariantPrice(product.id, v.id, {
            new_price: parseFloat(edit.price),
          } as UpdateVariantPriceRequest)
        );
      }
      if (parseInt(edit.newStock) !== v.stock && edit.stockReason) {
        ops.push(
          adminProductService.adjustStock(product.id, v.id, {
            new_stock: parseInt(edit.newStock),
            reason: edit.stockReason,
          } as AdjustStockRequest)
        );
      }
      if ((v.is_active ?? true) !== edit.is_active) {
        ops.push(
          adminProductService.setVariantActive(product.id, v.id, {
            is_active: edit.is_active,
          } as SetVariantActiveRequest)
        );
      }
      if (ops.length === 0) { flash(true, "ไม่มีการเปลี่ยนแปลง"); setSaving(false); return; }
      // ส่งทุก API call พร้อมกัน (parallel) — เร็วกว่าส่งทีละตัว
      await Promise.all(ops);
      flash(true, `บันทึก ${v.name} สำเร็จ`);
      setVariantEdits((prev) => ({ ...prev, [v.id]: { ...edit, stockReason: "" } }));
      onUpdated();
    } catch {
      flash(false, "บันทึก variant ไม่สำเร็จ");
    } finally {
      setSaving(false);
    }
  };

  const handleAddVariant = async () => {
    if (!product || !addForm.sku) return;
    setSaving(true);
    try {
      await adminProductService.addVariant(product.id, {
        sku: addForm.sku,
        name: addForm.name || addForm.sku,
        price: parseFloat(addForm.price) || 0,
        stock: parseInt(addForm.stock) || 0,
        attribute_value_ids: addForm.attribute_value_ids,
      } as AddVariantRequest);
      flash(true, "เพิ่ม variant สำเร็จ");
      setAddForm({ sku: "", name: "", price: "", stock: "", attribute_value_ids: [] });
      onUpdated();
    } catch {
      flash(false, "เพิ่ม variant ไม่สำเร็จ");
    } finally {
      setSaving(false);
    }
  };

  if (!open || !product) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/30 backdrop-blur-sm">
      <div className="relative w-full max-w-2xl rounded-2xl bg-white p-8 shadow-xl animate-in fade-in zoom-in-95 duration-200 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-bold text-on-surface">แก้ไขสินค้า</h2>
            <p className="text-xs text-secondary truncate max-w-xs">{product.name}</p>
          </div>
          <button onClick={onClose} className="rounded-full p-1.5 text-secondary hover:bg-surface-highest hover:text-on-surface">
            <X size={20} />
          </button>
        </div>

        {msg && (
          <div className={cn("mb-4 rounded-xl px-4 py-2.5 text-xs font-medium", msg.ok ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-600")}>
            {msg.text}
          </div>
        )}

        {/* ── General Info ── */}
        <div className="mb-6 rounded-xl border border-surface-highest p-5">
          <h3 className="mb-4 text-sm font-semibold text-on-surface">ข้อมูลทั่วไป</h3>
          <div className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-secondary">ชื่อสินค้า *</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-secondary">รายละเอียด</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
                className="w-full resize-none rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-secondary">หมวดหมู่</label>
                <select
                  value={categoryId}
                  onChange={(e) => setCategoryId(e.target.value ? Number(e.target.value) : "")}
                  className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
                >
                  <option value="">-- ไม่ระบุ --</option>
                  {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
                </select>
              </div>
              <div className="flex flex-col justify-end">
                <label className="mb-1 block text-xs font-medium text-secondary">สถานะสินค้า</label>
                <button
                  onClick={() => setIsActive((v) => !v)}
                  className={cn(
                    "flex items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-medium transition-colors",
                    isActive ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-600"
                  )}
                >
                  <span className={cn("h-2 w-2 rounded-full", isActive ? "bg-emerald-500" : "bg-red-400")} />
                  {isActive ? "Active" : "Inactive"}
                </button>
              </div>
            </div>
          </div>
          <div className="mt-4 flex justify-end">
            <button
              onClick={handleSaveGeneral}
              disabled={saving || !name}
              className="gradient-primary rounded-xl px-5 py-2 text-xs font-semibold text-white shadow-lg shadow-primary/25 disabled:opacity-50"
            >
              {saving ? "กำลังบันทึก..." : "บันทึกข้อมูลทั่วไป"}
            </button>
          </div>
        </div>

        {/* ── Variants ── */}
        <div className="mb-6 rounded-xl border border-surface-highest p-5">
          <h3 className="mb-4 text-sm font-semibold text-on-surface">Variants ({product.variants?.length ?? 0})</h3>
          <div className="space-y-4">
            {(product.variants ?? []).map((v) => {
              const e = variantEdits[v.id] ?? { price: String(v.price), newStock: String(v.stock), stockReason: "", is_active: true };
              return (
                <div key={v.id} className="rounded-xl bg-surface-low/30 p-4">
                  <div className="mb-3 flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-on-surface">{v.name}</p>
                      <p className="font-mono text-[10px] text-outline">{v.sku}</p>
                    </div>
                    <button
                      onClick={() => setVariantEdits((prev) => ({ ...prev, [v.id]: { ...e, is_active: !e.is_active } }))}
                      className={cn(
                        "rounded-full px-3 py-1 text-[10px] font-semibold uppercase",
                        e.is_active ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-500"
                      )}
                    >
                      {e.is_active ? "Active" : "Inactive"}
                    </button>
                  </div>
                  <div className="grid grid-cols-3 gap-3">
                    <div>
                      <label className="mb-1 block text-xs font-medium text-secondary">ราคา (฿)</label>
                      <input
                        type="number"
                        value={e.price}
                        onChange={(ev) => setVariantEdits((prev) => ({ ...prev, [v.id]: { ...e, price: ev.target.value } }))}
                        className="w-full rounded-xl bg-white px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary/20"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-secondary">สต็อกใหม่</label>
                      <input
                        type="number"
                        value={e.newStock}
                        onChange={(ev) => setVariantEdits((prev) => ({ ...prev, [v.id]: { ...e, newStock: ev.target.value } }))}
                        className="w-full rounded-xl bg-white px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary/20"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-secondary">เหตุผล (สต็อก)</label>
                      <input
                        type="text"
                        placeholder="เช่น รับสินค้าเพิ่ม"
                        value={e.stockReason}
                        onChange={(ev) => setVariantEdits((prev) => ({ ...prev, [v.id]: { ...e, stockReason: ev.target.value } }))}
                        className="w-full rounded-xl bg-white px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary/20"
                      />
                    </div>
                  </div>
                  <div className="mt-3 flex justify-end">
                    <button
                      onClick={() => handleSaveVariant(v)}
                      disabled={saving}
                      className="gradient-primary rounded-xl px-4 py-1.5 text-xs font-semibold text-white shadow-md disabled:opacity-50"
                    >
                      บันทึก variant
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* ── Add Variant ── */}
        <div className="rounded-xl border border-dashed border-surface-highest p-5">
          <h3 className="mb-4 text-sm font-semibold text-on-surface">เพิ่ม Variant ใหม่</h3>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-secondary">SKU *</label>
              <input type="text" placeholder="SKU-001" value={addForm.sku} onChange={(e) => setAddForm((f) => ({ ...f, sku: e.target.value }))}
                className="w-full rounded-xl bg-surface-low/40 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary/20" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-secondary">ชื่อ variant</label>
              <input type="text" placeholder="เช่น สีแดง / ไซส์ M" value={addForm.name} onChange={(e) => setAddForm((f) => ({ ...f, name: e.target.value }))}
                className="w-full rounded-xl bg-surface-low/40 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary/20" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-secondary">ราคา (฿)</label>
              <input type="number" placeholder="0.00" value={addForm.price} onChange={(e) => setAddForm((f) => ({ ...f, price: e.target.value }))}
                className="w-full rounded-xl bg-surface-low/40 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary/20" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-secondary">สต็อกเริ่มต้น</label>
              <input type="number" placeholder="0" value={addForm.stock} onChange={(e) => setAddForm((f) => ({ ...f, stock: e.target.value }))}
                className="w-full rounded-xl bg-surface-low/40 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary/20" />
            </div>
            <div className="col-span-2">
              <label className="mb-1 block text-xs font-medium text-secondary">Attribute Values</label>
              {attributes.length > 0 ? (
                <div className="space-y-2 rounded-xl bg-surface-low/40 p-3">
                  {attributes.map((attr) => (
                    <div key={attr.id}>
                      <p className="mb-1 text-xs font-semibold text-on-surface">{attr.name}</p>
                      <div className="flex flex-wrap gap-2">
                        {(attr.values ?? []).map((val) => {
                          const checked = addForm.attribute_value_ids.includes(val.id);
                          return (
                            <button
                              key={val.id}
                              type="button"
                              onClick={() =>
                                setAddForm((f) => ({
                                  ...f,
                                  attribute_value_ids: checked
                                    ? f.attribute_value_ids.filter((id) => id !== val.id)
                                    : [...f.attribute_value_ids, val.id],
                                }))
                              }
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
              ) : (
                <p className="text-xs text-outline">ยังไม่มี attributes — สร้างได้ที่หน้า Attributes</p>
              )}
            </div>
          </div>
          <div className="mt-4 flex justify-end">
            <button
              onClick={handleAddVariant}
              disabled={saving || !addForm.sku}
              className="gradient-primary rounded-xl px-5 py-2 text-xs font-semibold text-white shadow-lg shadow-primary/25 disabled:opacity-50"
            >
              {saving ? "กำลังเพิ่ม..." : "เพิ่ม Variant"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ════════  Products Page  ════════ */
// WHAT: หน้าหลัก admin — ตารางสินค้า + search + pagination
// HOW: load สินค้า/category/attribute ตอน mount → search debounce 300ms → reset page เมื่อ search เปลี่ยน
export default function AdminProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [attributes, setAttributes] = useState<Attribute[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [fetchError, setFetchError] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [imagesModalProduct, setImagesModalProduct] = useState<Product | null>(null);
  const [editProduct, setEditProduct] = useState<Product | null>(null);
  const limit = 10;

  //ดึงข้อมูล Master Data (ทำครั้งเดียวตอนโหลดหน้าเว็บ)
  useEffect(() => {
    categoryService.list().then((cats) => setCategories(cats)).catch(() => {});
    attributeService.list().then((attrs) => setAttributes(attrs)).catch(() => {});
  }, []);

  // fetchProducts ดึงสินค้าตาม page + search ปัจจุบัน
  // useCallback: ป้องกัน function re-create ทุก render → dependency ใน useEffect ไม่กระตุกซ้ำ
  const fetchProducts = useCallback(async () => {
    setLoading(true);
    setFetchError(false);
    try {
      const res = await productService.list({ page, limit, search: debouncedSearch || undefined });
      setProducts(res.items ?? []);
      setTotal(res.total ?? 0);
    } catch {
      setFetchError(true);
      setProducts([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, debouncedSearch]);

  useEffect(() => {
    fetchProducts();
  }, [fetchProducts]);

  // Debounce: หน่วง 300ms ก่อนยิง API
  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(t);
  }, [search]);

  // Reset กลับหน้า 1 เมื่อ search เปลี่ยน
  useEffect(() => {
    // เช็คก่อนว่าถ้าไม่ได้อยู่หน้า 1 ค่อยเซ็ต จะได้ไม่ทริกเกอร์ให้มันเรนเดอร์ซ้ำซ้อนฟรีๆ
    if (page !== 1) {
      setPage(1);
    }
  }, [debouncedSearch]);

  const handleDelete = async (id: number) => {
    if (!confirm("ต้องการลบสินค้านี้หรือไม่?")) return;
    await adminProductService.delete(id);
    fetchProducts();
  };

  const filtered = products;

  const totalPages = Math.ceil(total / limit) || 1;

  /* helper */
  const getMinPrice = (p: Product) =>
    p.variants?.length ? Math.min(...p.variants.map((v) => v.price)) : 0;
  const getTotalStock = (p: Product) =>
    p.variants?.reduce((s, v) => s + v.stock, 0) ?? 0;
  const getPaginationGroup = () => {
    const MAX_BUTTON = 5;
    let strat = Math.max(1, page - 2);
    let end = Math.min(totalPages, strat + MAX_BUTTON - 1);

    if (end - strat + 1 < MAX_BUTTON ) {
      strat = Math.max(1, end - MAX_BUTTON + 1);
    }
    
    return Array.from({ length: end - strat + 1 }, (_, i) => strat + i);
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-on-surface">
            จัดการสินค้า
          </h1>
          <p className="text-sm text-secondary">
            ทั้งหมด {total} รายการ
          </p>
        </div>
        <button
          onClick={() => setModalOpen(true)}
          className="gradient-primary flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl active:scale-95"
        >
          <Plus size={18} />
          เพิ่มสินค้า
        </button>
      </div>

      {/* Filter bar */}
      <div className="mb-6 flex items-center gap-3 rounded-2xl bg-white p-4 shadow-ambient">
        <div className="relative flex-1 max-w-sm">
          <Search
            size={16}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-outline"
          />
          <input
            type="text"
            placeholder="ค้นหาสินค้า..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg bg-surface-highest py-2 pl-9 pr-4 text-sm outline-none transition-all focus:ring-2 focus:ring-primary/20"
          />
        </div>
        {/* TODO: filter เหล่านี้ยังเป็น UI-only — ยังไม่ได้ connect กับ fetchProducts */}
        <select className="rounded-lg bg-surface-highest px-3 py-2 text-sm text-secondary outline-none">
          <option>ทุกหมวดหมู่</option>
          {categories.map((c) => (
            <option key={c.id}>{c.name}</option>
          ))}
        </select>
        <select className="rounded-lg bg-surface-highest px-3 py-2 text-sm text-secondary outline-none">
          <option>ทุกสถานะ</option>
          <option>Active</option>
          <option>Inactive</option>
        </select>
      </div>

      {/* Error banner */}
      {fetchError && (
        <div className="mb-4 flex items-center gap-3 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600">
          <span className="font-semibold">เกิดข้อผิดพลาด</span>
          <span className="text-xs">ไม่สามารถโหลดข้อมูลสินค้าได้ — กรุณาตรวจสอบการเชื่อมต่อหรือลองใหม่</span>
          <button
            onClick={fetchProducts}
            className="ml-auto rounded-lg bg-red-100 px-3 py-1 text-xs font-medium hover:bg-red-200"
          >
            ลองใหม่
          </button>
        </div>
      )}

      {/* Table */}
      <div className="overflow-hidden rounded-2xl bg-white shadow-ambient">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-surface-highest bg-surface-low/30 text-xs uppercase tracking-wider text-secondary">
                <th className="px-6 py-4 font-medium">สินค้า</th>
                <th className="px-4 py-4 font-medium">SKU</th>
                <th className="px-4 py-4 font-medium">หมวดหมู่</th>
                <th className="px-4 py-4 font-medium">ราคา</th>
                <th className="px-4 py-4 font-medium">สต็อก</th>
                <th className="px-4 py-4 font-medium">สถานะ</th>
                <th className="px-4 py-4 font-medium text-right">จัดการ</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-secondary">
                    กำลังโหลด...
                  </td>
                </tr>
              ) : filtered.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-secondary">
                    <Package size={40} className="mx-auto mb-2 text-outline" />
                    ไม่พบสินค้า
                  </td>
                </tr>
              ) : (
                filtered.map((p) => (
                  <tr
                    key={p.id}
                    className="border-b border-surface-highest/60 transition-colors hover:bg-surface-low/40"
                  >
                    {/* Product */}
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        {p.image_urls?.[0] ? (
                          <img
                            src={p.image_urls[0]}
                            alt={p.name}
                            className="h-10 w-10 rounded-lg bg-surface-highest object-cover"
                          />
                        ) : (
                          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-surface-highest text-outline">
                            <Package size={18} />
                          </div>
                        )}
                        <span className="font-medium text-on-surface">
                          {truncate(p.name, 35)}
                        </span>
                      </div>
                    </td>
                    {/* SKU */}
                    <td className="px-4 py-4 font-mono text-xs text-secondary">
                      {p.variants?.[0]?.sku ?? "-"}
                    </td>
                    {/* Category */}
                    <td className="px-4 py-4">
                      {p.categories?.[0] ? (
                        <span className="rounded-full bg-primary-container/40 px-2.5 py-0.5 text-[10px] font-semibold text-primary">
                          {p.categories[0].name}
                        </span>
                      ) : (
                        <span className="text-xs text-outline">—</span>
                      )}
                    </td>
                    {/* Price */}
                    <td className="px-4 py-4 font-medium text-on-surface">
                      {formatBaht(getMinPrice(p))}
                    </td>
                    {/* Stock */}
                    <td className="px-4 py-4">
                      <StockBar stock={getTotalStock(p)} />
                    </td>
                    {/* Status */}
                    <td className="px-4 py-4">
                      <ActiveBadge active={p.is_active} />
                    </td>
                    {/* Actions */}
                    <td className="px-4 py-4 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => setImagesModalProduct(p)}
                          className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest hover:text-primary"
                          title="จัดการรูปภาพ"
                        >
                          <Images size={15} />
                        </button>
                        <button
                          onClick={() => setEditProduct(p)}
                          className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest hover:text-primary"
                          title="แก้ไขสินค้า"
                        >
                          <Pencil size={15} />
                        </button>
                        <button
                          onClick={() => handleDelete(p.id)}
                          className="rounded-lg p-2 text-secondary transition-colors hover:bg-red-50 hover:text-error"
                        >
                          <Trash2 size={15} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between border-t border-surface-highest px-6 py-4">
          <p className="text-xs text-secondary">
            แสดง {(page - 1) * limit + 1}–{Math.min(page * limit, total)} จาก{" "}
            {total} รายการ
          </p>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
              className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest disabled:opacity-40"
            >
              <ChevronLeft size={16} />
            </button>
            {getPaginationGroup().map(
              (n) => (
                <button
                  key={n}
                  onClick={() => setPage(n)}
                  className={cn(
                    "h-8 w-8 rounded-lg text-xs font-medium transition-colors",
                    n === page
                      ? "bg-primary text-white"
                      : "text-secondary hover:bg-surface-highest"
                  )}
                >
                  {n}
                </button>
              )
            )}
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
              className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest disabled:opacity-40"
            >
              <ChevronRight size={16} />
            </button>
          </div>
        </div>
      </div>

      {/* Modal */}
      <AddProductModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        categories={categories}
        attributes={attributes}
        onCreated={fetchProducts}
      />
      <EditProductModal
        open={!!editProduct}
        onClose={() => setEditProduct(null)}
        product={editProduct}
        categories={categories}
        attributes={attributes}
        onUpdated={() => { fetchProducts(); setEditProduct(null); }}
      />
      <ManageImagesModal
        open={!!imagesModalProduct}
        onClose={() => setImagesModalProduct(null)}
        product={imagesModalProduct}
        onUpdated={fetchProducts}
      />
    </div>
  );
}
