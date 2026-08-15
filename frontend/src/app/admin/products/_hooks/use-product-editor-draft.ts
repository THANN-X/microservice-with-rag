/**
 * use-product-editor-draft.ts — สมองของ modal แก้ไขสินค้า (ไม่มี JSX เลย)
 *
 * What: เก็บค่าที่ผู้ใช้แก้เป็น draft → เทียบกับ product เดิมว่าอะไรเปลี่ยน →
 *       แปลงส่วนที่เปลี่ยนเป็น "op" (1 op = 1 API call) แล้วยิงพร้อมกันทีเดียว
 * Why:  ตรรกะส่วนนี้คือหัวใจของหน้าแก้ไขสินค้า (โดยเฉพาะการกันยิง adjustStock ซ้ำ
 *       ซึ่งถ้าพลาดจะสร้างประวัติปรับสต็อกซ้ำในระบบ) — แยกออกจาก JSX เพื่อให้
 *       อ่านและเทสได้โดยไม่ต้อง render
 * How:  ปุ่มบันทึกปุ่มเดียวเรียก save() → คืน SaveResult ให้ฝั่ง UI เอาไปแสดงข้อความ
 */
"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { adminProductService } from "@/lib/services";
import { toUserMessage } from "@/lib/errors";
import type {
  Product,
  Variant,
  AdjustStockRequest,
  SetProductActiveRequest,
  SetVariantActiveRequest,
  UpdateProductGeneralInfoRequest,
  UpdateVariantPriceRequest,
} from "@/lib/types";

export type EditorTab = "general" | "images" | "variants";

export type VariantDraft = {
  price: string;
  newStock: string;
  stockReason: string;
  is_active: boolean;
  images: string[];
};

/** op หนึ่งชิ้น = API call หนึ่งครั้ง — key ใช้กันยิงซ้ำตอน retry, label ใช้บอกผู้ใช้ว่าอันไหนพัง */
type SaveOp = { key: string; label: string; run: () => Promise<unknown> };

export type VariantDiff = {
  v: Variant;
  draft?: VariantDraft;
  price: boolean;
  stock: boolean;
  active: boolean;
  images: boolean;
  /** แก้สต็อกแล้วแต่ยังไม่กรอกเหตุผล — backend บังคับ */
  needsReason: boolean;
  /** ช่องราคาว่าง / ไม่ใช่ตัวเลข / ไม่เกิน 0 */
  priceInvalid: boolean;
  /** ช่องสต็อกว่าง / ไม่ใช่ตัวเลข / ไม่เกิน 0 */
  stockInvalid: boolean;
};

export type SaveResult =
  | { status: "blocked"; tab: EditorTab; text: string }
  | { status: "noop" }
  | { status: "ok"; count: number }
  | { status: "error"; text: string };

const sameUrls = (a: string[], b: string[]) =>
  a.length === b.length && a.every((u, i) => u === b[i]);

export function useProductEditorDraft(product: Product | null) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState<number | "">("");
  const [isActive, setIsActive] = useState(true);
  const [productImages, setProductImages] = useState<string[]>([]);
  const [drafts, setDrafts] = useState<Record<number, VariantDraft>>({});
  // op ที่สำเร็จไปแล้วตอน save รอบก่อน — กันยิงซ้ำถ้าผู้ใช้กดบันทึกอีกรอบหลังบางตัวพัง
  // (สำคัญกับ adjustStock เพราะยิงซ้ำ = สร้างประวัติปรับสต็อกซ้ำ)
  const [doneOps, setDoneOps] = useState<Set<string>>(new Set());
  const [saving, setSaving] = useState(false);

  const variants = useMemo(() => product?.variants ?? [], [product]);

  /* ── โหลดค่าเริ่มต้นจาก product ทุกครั้งที่เปลี่ยนตัวที่แก้ ── */
  useEffect(() => {
    if (!product) return;
    setName(product.name);
    setDescription(product.description ?? "");
    setCategoryId(product.categories?.[0]?.id ?? "");
    setIsActive(product.is_active ?? true);
    setProductImages(product.image_urls ?? []);
    const init: Record<number, VariantDraft> = {};
    (product.variants ?? []).forEach((v) => {
      init[v.id] = {
        price: String(v.price),
        newStock: String(v.stock),
        stockReason: "",
        is_active: v.is_active ?? true,
        images: v.image_urls ?? [],
      };
    });
    setDrafts(init);
    setDoneOps(new Set());
  }, [product]);

  const setDraft = useCallback(
    (id: number, patch: Partial<VariantDraft>) =>
      setDrafts((prev) => ({ ...prev, [id]: { ...prev[id], ...patch } })),
    []
  );

  /* ── diff: อะไรเปลี่ยนไปบ้าง ── */
  const generalChanged = product ? name !== product.name || description !== (product.description ?? "") || categoryId !== (product.categories?.[0]?.id ?? "") : false;
  const activeChanged = product ? isActive !== (product.is_active ?? true) : false;
  const productImagesChanged = product ? !sameUrls(productImages, product.image_urls ?? []) : false;

  const variantDiff: VariantDiff[] = useMemo(
    () =>
      variants.map((v) => {
        const d = drafts[v.id];
        if (!d) {
          return {
            v,
            price: false, stock: false, active: false, images: false,
            needsReason: false, priceInvalid: false, stockInvalid: false,
          };
        }
        // ช่องว่างจะ parse ได้ NaN — ต้องนับว่า "เปลี่ยนแล้ว" (ผู้ใช้ลบค่าทิ้งจริง)
        // แต่ห้ามปล่อยให้ยิง API ไม่งั้น new_stock/new_price จะกลายเป็น null ใน JSON
        const priceNum = parseFloat(d.price);
        const stockNum = parseInt(d.newStock);
        const price = priceNum !== v.price;
        const stock = stockNum !== v.stock;
        return {
          v,
          draft: d,
          price,
          stock,
          active: (v.is_active ?? true) !== d.is_active,
          images: !sameUrls(d.images, v.image_urls ?? []),
          needsReason: stock && !d.stockReason.trim(),
          // เขียนเป็น !(x > 0) ไม่ใช่ x <= 0 เพราะ NaN <= 0 เป็น false — ค่าว่างจะรอดด่านไปได้
          priceInvalid: price && !(priceNum > 0),
          stockInvalid: stock && !(stockNum > 0),
        };
      }),
    [variants, drafts]
  );

  /* ── ตัวเลข badge บนแท็บ ── */
  const generalCount = (generalChanged ? 1 : 0) + (activeChanged ? 1 : 0);
  const imagesCount =
    (productImagesChanged ? 1 : 0) + variantDiff.filter((d) => d.images).length;
  const variantsCount = variantDiff.filter((d) => d.price || d.stock || d.active).length;
  const totalChanges = generalCount + imagesCount + variantsCount;

  /* ── validation ที่ backend บังคับ (ดู UpdateProductGeneralInfoReq / AdjustStockReq) ── */
  const blockers: { tab: EditorTab; text: string }[] = [];
  // เช็คเฉพาะตอนที่จะส่ง updateGeneralInfo จริง — ถ้าผู้ใช้แก้แค่ราคา variant
  // ก็ไม่ควรโดนบล็อกเพราะสินค้าเก่าเผอิญมี description ว่าง
  if (generalChanged) {
    if (!name.trim()) blockers.push({ tab: "general", text: "ต้องกรอกชื่อสินค้า" });
    if (!description.trim()) blockers.push({ tab: "general", text: "ต้องกรอกรายละเอียดสินค้า" });
    if (description.length > 255)
      blockers.push({ tab: "general", text: "รายละเอียดต้องไม่เกิน 255 ตัวอักษร" });
    if (categoryId === "")
      blockers.push({ tab: "general", text: "ต้องเลือกหมวดหมู่อย่างน้อย 1 รายการ" });
  }
  variantDiff.forEach((d) => {
    if (d.needsReason)
      blockers.push({ tab: "variants", text: `ต้องระบุเหตุผลที่แก้สต็อกของ "${d.v.name}"` });
    if (d.stockInvalid)
      blockers.push({ tab: "variants", text: `สต็อกของ "${d.v.name}" ต้องเป็นตัวเลขมากกว่า 0` });
    if (d.priceInvalid)
      blockers.push({ tab: "variants", text: `ราคาของ "${d.v.name}" ต้องเป็นตัวเลขมากกว่า 0` });
  });

  /** แปลงสิ่งที่เปลี่ยนเป็นรายการ API call — ตัด op ที่สำเร็จไปแล้วรอบก่อนออก */
  const buildOps = (): SaveOp[] => {
    if (!product) return [];
    const ops: SaveOp[] = [];
    if (generalChanged) {
      ops.push({
        key: "general",
        label: "ข้อมูลทั่วไป",
        run: () =>
          adminProductService.updateGeneralInfo(product.id, {
            name: name.trim(),
            description: description.trim(),
            category_ids: [Number(categoryId)],
          } as UpdateProductGeneralInfoRequest),
      });
    }
    if (activeChanged) {
      ops.push({
        key: "active",
        label: "สถานะสินค้า",
        run: () =>
          adminProductService.setProductActive(product.id, {
            is_active: isActive,
          } as SetProductActiveRequest),
      });
    }
    if (productImagesChanged) {
      ops.push({
        key: "images",
        label: "รูปภาพสินค้า",
        run: () =>
          adminProductService.updateProductImages(product.id, { image_urls: productImages }),
      });
    }
    variantDiff.forEach((d) => {
      const draft = d.draft;
      if (!draft) return;
      if (d.price) {
        ops.push({
          key: `price-${d.v.id}`,
          label: `ราคา "${d.v.name}"`,
          run: () =>
            adminProductService.updateVariantPrice(product.id, d.v.id, {
              new_price: parseFloat(draft.price),
            } as UpdateVariantPriceRequest),
        });
      }
      if (d.stock) {
        ops.push({
          key: `stock-${d.v.id}`,
          label: `สต็อก "${d.v.name}"`,
          run: () =>
            adminProductService.adjustStock(product.id, d.v.id, {
              new_stock: parseInt(draft.newStock),
              reason: draft.stockReason.trim(),
            } as AdjustStockRequest),
        });
      }
      if (d.active) {
        ops.push({
          key: `vactive-${d.v.id}`,
          label: `สถานะ "${d.v.name}"`,
          run: () =>
            adminProductService.setVariantActive(product.id, d.v.id, {
              is_active: draft.is_active,
            } as SetVariantActiveRequest),
        });
      }
      if (d.images) {
        ops.push({
          key: `vimages-${d.v.id}`,
          label: `รูปภาพ "${d.v.name}"`,
          run: () =>
            adminProductService.updateVariantImages(product.id, d.v.id, {
              image_urls: draft.images,
            }),
        });
      }
    });
    return ops.filter((o) => !doneOps.has(o.key));
  };

  const save = async (): Promise<SaveResult> => {
    if (!product) return { status: "noop" };
    if (blockers.length > 0) {
      // คืน tab ที่ติดปัญหาไปด้วย ฝั่ง UI จะได้พาผู้ใช้ไปแท็บนั้นเลย ไม่ต้องให้ไล่หาเอง
      return { status: "blocked", tab: blockers[0].tab, text: blockers[0].text };
    }
    const ops = buildOps();
    if (ops.length === 0) return { status: "noop" };

    setSaving(true);
    // allSettled (ไม่ใช่ all): ถ้ามีบางตัวพัง ต้องรู้ว่าตัวไหนสำเร็จตัวไหนไม่สำเร็จ
    // ไม่งั้นจะรายงานว่า "บันทึกไม่สำเร็จ" ทั้งที่ 4 ใน 5 ผ่านไปแล้ว
    const results = await Promise.allSettled(ops.map((o) => o.run()));
    const failed = ops.filter((_, i) => results[i].status === "rejected");
    const succeeded = ops.filter((_, i) => results[i].status === "fulfilled");
    setSaving(false);

    if (failed.length === 0) return { status: "ok", count: ops.length };

    // จำตัวที่สำเร็จไว้ กดบันทึกซ้ำจะได้ยิงเฉพาะตัวที่พัง
    setDoneOps((prev) => new Set([...prev, ...succeeded.map((o) => o.key)]));
    const firstErr = results.find((r) => r.status === "rejected") as
      | PromiseRejectedResult
      | undefined;
    return {
      status: "error",
      text:
        succeeded.length > 0
          ? `บันทึกสำเร็จ ${succeeded.length} รายการ แต่ไม่สำเร็จ: ${failed
              .map((o) => o.label)
              .join(", ")}`
          : toUserMessage(
              firstErr?.reason,
              `บันทึกไม่สำเร็จ: ${failed.map((o) => o.label).join(", ")}`
            ),
    };
  };

  return {
    // ค่าในฟอร์ม
    name, setName,
    description, setDescription,
    categoryId, setCategoryId,
    isActive, setIsActive,
    productImages, setProductImages,
    drafts, setDraft,
    // ผลการเทียบกับของเดิม
    variants,
    variantDiff,
    productImagesChanged,
    generalCount,
    imagesCount,
    variantsCount,
    totalChanges,
    // การบันทึก
    saving,
    save,
  };
}
