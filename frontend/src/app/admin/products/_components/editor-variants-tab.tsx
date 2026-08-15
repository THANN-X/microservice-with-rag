// WHAT: แท็บ "Variants" ของ modal แก้ไขสินค้า — แก้ราคา/สต็อก/สถานะ ราย variant + เพิ่ม variant ใหม่
"use client";

import { AlertCircle } from "lucide-react";
import { ToggleTrack } from "@/components/admin/toggle-track";
import { fieldLabel, inputFieldOnCard } from "@/components/admin/form-styles";
import { cn } from "@/lib/utils";
import { VariantOptions } from "./product-badges";
import { AddVariantForm } from "./add-variant-form";
import type { VariantDiff, VariantDraft } from "../_hooks/use-product-editor-draft";
import type { Attribute } from "@/lib/types";

export function EditorVariantsTab({
  productId,
  variantDiff,
  onDraftChange,
  attributes,
  onVariantAdded,
  onVariantAddError,
}: {
  productId: number;
  variantDiff: VariantDiff[];
  onDraftChange: (variantId: number, patch: Partial<VariantDraft>) => void;
  attributes: Attribute[];
  onVariantAdded: (message: string) => void;
  onVariantAddError: (message: string) => void;
}) {
  return (
    <div className="space-y-4">
      {variantDiff.map(({
        v, draft,
        price: priceChanged, stock: stockChanged, active: activeChanged,
        needsReason, priceInvalid, stockInvalid,
      }) => {
        if (!draft) return null;
        return (
          <div key={v.id} className="rounded-xl bg-surface-low/30 p-4">
            <div className="mb-3 flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-sm font-medium text-on-surface">{v.name}</p>
                <p className="font-mono text-[10px] text-outline">{v.sku}</p>
                <VariantOptions options={v.options} className="mt-1.5" />
              </div>
              <button
                role="switch"
                aria-checked={draft.is_active}
                onClick={() => onDraftChange(v.id, { is_active: !draft.is_active })}
                className="flex shrink-0 items-center gap-2 rounded-lg px-2 py-1 transition-colors hover:bg-white"
              >
                <ToggleTrack on={draft.is_active} size="sm" />
                <span
                  className={cn(
                    "text-[11px] font-medium",
                    draft.is_active ? "text-emerald-700" : "text-secondary"
                  )}
                >
                  {draft.is_active ? "Active" : "Inactive"}
                </span>
                {activeChanged && <span className="text-[10px] text-amber-600">•</span>}
              </button>
            </div>

            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className={fieldLabel}>
                  ราคา (฿)
                  {priceChanged && (
                    <span className="ml-1 font-normal text-outline">(เดิม {v.price})</span>
                  )}
                </label>
                <input
                  type="number"
                  min={0.01}
                  step="0.01"
                  value={draft.price}
                  onChange={(e) => onDraftChange(v.id, { price: e.target.value })}
                  className={cn(
                    inputFieldOnCard,
                    priceInvalid
                      ? "ring-2 ring-error/40"
                      : priceChanged && "ring-2 ring-amber-300"
                  )}
                />
              </div>
              <div>
                <label className={fieldLabel}>
                  สต็อกใหม่
                  {stockChanged && (
                    <span className="ml-1 font-normal text-outline">(เดิม {v.stock})</span>
                  )}
                </label>
                <input
                  type="number"
                  min={1}
                  step={1}
                  value={draft.newStock}
                  onChange={(e) => onDraftChange(v.id, { newStock: e.target.value })}
                  className={cn(
                    inputFieldOnCard,
                    stockInvalid
                      ? "ring-2 ring-error/40"
                      : stockChanged && "ring-2 ring-amber-300"
                  )}
                />
              </div>
              <div>
                <label className={fieldLabel}>
                  เหตุผล (สต็อก){stockChanged && <span className="text-error"> *</span>}
                </label>
                <input
                  type="text"
                  placeholder={stockChanged ? "จำเป็นต้องกรอก" : "เช่น รับสินค้าเพิ่ม"}
                  value={draft.stockReason}
                  onChange={(e) => onDraftChange(v.id, { stockReason: e.target.value })}
                  className={cn(
                    inputFieldOnCard,
                    needsReason && "ring-2 ring-error/40 placeholder:text-error/60"
                  )}
                />
              </div>
            </div>

            {(priceInvalid || stockInvalid) && (
              <p className="mt-2 flex items-center gap-1.5 text-[11px] font-medium text-error">
                <AlertCircle size={12} />
                {priceInvalid && stockInvalid
                  ? "ราคาและสต็อกต้องเป็นตัวเลขมากกว่า 0"
                  : priceInvalid
                    ? "ราคาต้องเป็นตัวเลขมากกว่า 0"
                    : "สต็อกต้องเป็นตัวเลขมากกว่า 0"}
              </p>
            )}

            {needsReason && (
              <p className="mt-2 flex items-center gap-1.5 text-[11px] font-medium text-error">
                <AlertCircle size={12} />
                แก้สต็อกแล้วต้องระบุเหตุผล — ระบบเก็บเป็นประวัติการปรับสต็อก
              </p>
            )}
          </div>
        );
      })}

      <AddVariantForm
        productId={productId}
        attributes={attributes}
        onAdded={onVariantAdded}
        onError={onVariantAddError}
      />
    </div>
  );
}
