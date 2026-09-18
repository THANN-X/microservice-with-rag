// WHAT: modal เดียวจบสำหรับแก้ไขสินค้า — รวม "แก้ไขสินค้า" กับ "จัดการรูปภาพ" ที่เคยแยกกัน
// WHY:  เดิมต้องเปิด-ปิด modal สองตัวเพื่อแก้สินค้าตัวเดียว และมีปุ่มบันทึกกระจาย 3-4 ปุ่ม
//       ถ้าลืมกดปุ่มใดปุ่มหนึ่งแล้วปิด ข้อมูลหายเงียบ
// HOW:  ตรรกะ draft/diff/save อยู่ใน useProductEditorDraft — ไฟล์นี้เหลือแค่ layout
//       กับการแปลงผลลัพธ์ของ save() เป็นข้อความบอกผู้ใช้
"use client";

import { useCallback, useEffect, useState } from "react";
import { ModalShell } from "@/components/admin/modal-shell";
import { ConfirmDialog } from "@/components/admin/confirm-dialog";
import { FormMessage } from "@/components/admin/form-message";
import { TabBar } from "@/components/admin/tab-bar";
import { useProductEditorDraft, type EditorTab } from "../_hooks/use-product-editor-draft";
import { EditorGeneralTab } from "./editor-general-tab";
import { EditorImagesTab } from "./editor-images-tab";
import { EditorVariantsTab } from "./editor-variants-tab";
import type { Attribute, Product } from "@/lib/types";

export function ProductEditorModal({
  open,
  product,
  initialTab,
  categories,
  attributes,
  onClose,
  onSaved,
}: {
  open: boolean;
  product: Product | null;
  initialTab: EditorTab;
  categories: { id: number; label: string }[];
  attributes: Attribute[];
  onClose: () => void;
  /** เรียกเมื่อบันทึกสำเร็จ — page จะ refetch, ปิด modal แล้วโชว์ banner ข้อความนี้ */
  onSaved: (message: string) => void;
}) {
  const draft = useProductEditorDraft(product);
  const [tab, setTab] = useState<EditorTab>(initialTab);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  useEffect(() => {
    if (open) setTab(initialTab);
  }, [open, initialTab]);

  // เปลี่ยนสินค้าที่กำลังแก้ → ล้างข้อความของสินค้าตัวก่อนออก
  useEffect(() => setMsg(null), [product]);

  const { totalChanges } = draft;
  const requestClose = useCallback(() => {
    if (totalChanges > 0) {
      setShowCloseConfirm(true);
      return;
    }
    onClose();
  }, [totalChanges, onClose]);

  const handleSaveAll = async () => {
    const result = await draft.save();
    switch (result.status) {
      case "blocked":
        // พาไปแท็บที่ติดปัญหาให้เลย ไม่ต้องให้ผู้ใช้ไล่หาเอง
        setTab(result.tab);
        setMsg({ ok: false, text: result.text });
        break;
      case "noop":
        setMsg({ ok: true, text: "ไม่มีการเปลี่ยนแปลง" });
        break;
      case "ok":
        onSaved(`บันทึกการเปลี่ยนแปลง ${result.count} รายการเรียบร้อย`);
        break;
      case "error":
        setMsg({ ok: false, text: result.text });
        break;
    }
  };

  if (!product) return null;

  return (
    <>
      <ModalShell
        open={open}
        onClose={requestClose}
        size="3xl"
        title="แก้ไขสินค้า"
        subtitle={product.name}
        tabs={
          <TabBar
            active={tab}
            onChange={setTab}
            tabs={[
              { key: "general" as EditorTab, label: "ข้อมูลทั่วไป", badge: draft.generalCount },
              { key: "images" as EditorTab, label: "รูปภาพ", badge: draft.imagesCount },
              {
                key: "variants" as EditorTab,
                label: `Variants (${draft.variants.length})`,
                badge: draft.variantsCount,
              },
            ]}
          />
        }
        footer={
          <div className="space-y-3">
            {msg && <FormMessage ok={msg.ok} text={msg.text} />}
            <div className="flex items-center justify-between gap-4">
              <p className="text-xs text-secondary">
                {totalChanges > 0 ? (
                  <span className="font-medium text-amber-600">
                    มีการเปลี่ยนแปลง {totalChanges} รายการที่ยังไม่บันทึก
                  </span>
                ) : (
                  "ยังไม่มีการเปลี่ยนแปลง"
                )}
              </p>
              <div className="flex gap-3">
                <button
                  onClick={requestClose}
                  className="rounded-xl px-5 py-2.5 text-sm font-medium text-secondary hover:bg-surface-highest"
                >
                  ปิด
                </button>
                <button
                  onClick={handleSaveAll}
                  disabled={draft.saving || totalChanges === 0}
                  className="gradient-primary rounded-xl px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl disabled:opacity-50"
                >
                  {draft.saving ? "กำลังบันทึก..." : "บันทึกการเปลี่ยนแปลง"}
                </button>
              </div>
            </div>
          </div>
        }
      >
        {tab === "general" && (
          <EditorGeneralTab
            name={draft.name}
            onNameChange={draft.setName}
            description={draft.description}
            onDescriptionChange={draft.setDescription}
            categoryId={draft.categoryId}
            onCategoryChange={draft.setCategoryId}
            categories={categories}
            isActive={draft.isActive}
            onActiveChange={draft.setIsActive}
          />
        )}

        {tab === "images" && (
          <EditorImagesTab
            productImages={draft.productImages}
            onProductImagesChange={draft.setProductImages}
            productImagesChanged={draft.productImagesChanged}
            variantDiff={draft.variantDiff}
            onVariantImagesChange={(id, urls) => draft.setDraft(id, { images: urls })}
          />
        )}

        {tab === "variants" && (
          <EditorVariantsTab
            // key: เปลี่ยนสินค้าแล้วฟอร์มเพิ่ม variant ต้องเริ่มใหม่ ไม่ค้างค่าของตัวก่อน
            key={product.id}
            productId={product.id}
            variantDiff={draft.variantDiff}
            onDraftChange={draft.setDraft}
            attributes={attributes}
            onVariantAdded={onSaved}
            onVariantAddError={(text) => setMsg({ ok: false, text })}
          />
        )}
      </ModalShell>

      <ConfirmDialog
        open={showCloseConfirm}
        title="ปิดโดยไม่บันทึก?"
        message={`มีการเปลี่ยนแปลง ${totalChanges} รายการที่ยังไม่ได้บันทึก ถ้าปิดตอนนี้การแก้ไขทั้งหมดจะหายไป`}
        confirmLabel="ปิดโดยไม่บันทึก"
        cancelLabel="กลับไปแก้ไข"
        danger
        onConfirm={() => {
          setShowCloseConfirm(false);
          onClose();
        }}
        onCancel={() => setShowCloseConfirm(false)}
      />
    </>
  );
}
