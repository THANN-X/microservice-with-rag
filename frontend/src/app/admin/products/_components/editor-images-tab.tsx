// WHAT: แท็บ "รูปภาพ" ของ modal แก้ไขสินค้า — รูปหลักของสินค้า + รูปแยกราย variant
"use client";

import { CloudinaryImageUploader } from "@/components/admin/cloudinary-uploader";
import { VariantOptions } from "./product-badges";
import type { VariantDiff } from "../_hooks/use-product-editor-draft";

function ChangedTag() {
  return <span className="ml-2 text-[10px] font-medium text-amber-600">แก้ไขแล้ว</span>;
}

export function EditorImagesTab({
  productImages,
  onProductImagesChange,
  productImagesChanged,
  variantDiff,
  onVariantImagesChange,
}: {
  productImages: string[];
  onProductImagesChange: (urls: string[]) => void;
  productImagesChanged: boolean;
  variantDiff: VariantDiff[];
  onVariantImagesChange: (variantId: number, urls: string[]) => void;
}) {
  return (
    <div className="space-y-6">
      <div className="rounded-xl border border-surface-highest p-4">
        <h3 className="mb-1 text-sm font-semibold text-on-surface">
          รูปภาพหลักของสินค้า
          {productImagesChanged && <ChangedTag />}
        </h3>
        <p className="mb-3 text-[10px] text-outline">รูปแรกใช้เป็นรูปหน้าปกในตารางและหน้าร้าน</p>
        <CloudinaryImageUploader urls={productImages} onChange={onProductImagesChange} />
      </div>

      {variantDiff.map((d) => (
        <div key={d.v.id} className="rounded-xl border border-surface-highest p-4">
          <h3 className="mb-1 text-sm font-semibold text-on-surface">
            {d.v.name}
            {d.images && <ChangedTag />}
          </h3>
          <p className="font-mono text-[10px] text-outline">{d.v.sku}</p>
          <VariantOptions options={d.v.options} className="mb-3 mt-1.5" />
          <CloudinaryImageUploader
            urls={d.draft?.images ?? []}
            onChange={(urls) => onVariantImagesChange(d.v.id, urls)}
          />
        </div>
      ))}
    </div>
  );
}
