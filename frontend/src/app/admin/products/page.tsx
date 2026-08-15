// WHAT: หน้า Admin จัดการสินค้า — ตารางสินค้า + filter + pagination + modal เพิ่ม/แก้ไข
// โครงสร้าง:
//   _hooks/use-product-list        — โหลดสินค้า/หมวดหมู่/attribute, debounce search, แบ่งหน้า
//   _hooks/use-product-editor-draft — draft/diff/save ของ modal แก้ไข (ไม่มี JSX)
//   _components/*                  — ตาราง, ตัวกรอง, modal เพิ่มสินค้า, modal แก้ไข (3 แท็บ)
//   components/admin/*             — ชิ้นส่วนที่ใช้ร่วมกับหน้า admin อื่น (modal, dialog, uploader ฯลฯ)
// ไฟล์นี้เหลือหน้าที่เดียว: ต่อชิ้นส่วนเข้าด้วยกัน + จัดการ banner แจ้งผลและการลบสินค้า
"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { AlertCircle, Plus, X } from "lucide-react";
import { ConfirmDialog } from "@/components/admin/confirm-dialog";
import { FormMessage } from "@/components/admin/form-message";
import { PaginationBar } from "@/components/admin/pagination-bar";
import { adminProductService } from "@/lib/services";
import { toUserMessage } from "@/lib/errors";
import type { Product } from "@/lib/types";
import { useProductList } from "./_hooks/use-product-list";
import type { EditorTab } from "./_hooks/use-product-editor-draft";
import { ProductFilters } from "./_components/product-filters";
import { ProductTable } from "./_components/product-table";
import { AddProductModal } from "./_components/add-product-modal";
import { ProductEditorModal } from "./_components/product-editor-modal";

export default function AdminProductsPage() {
  const list = useProductList();

  const [addOpen, setAddOpen] = useState(false);
  // editTarget เก็บทั้งสินค้าและแท็บที่จะเปิด — ปุ่มรูปภาพในตารางจึงกระโดดเข้าแท็บรูปได้เลย
  const [editTarget, setEditTarget] = useState<{ product: Product; tab: EditorTab } | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<Product | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const [successMsg, setSuccessMsg] = useState("");
  const successTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // banner แจ้งผลสำเร็จ — modal ปิดไปแล้วตอนบันทึกเสร็จ ข้อความจึงต้องมาโผล่ที่หน้าหลัก
  const flashSuccess = useCallback((text: string) => {
    if (successTimerRef.current) clearTimeout(successTimerRef.current);
    // WHY: deleteError ไม่มี auto-dismiss — ถ้าไม่ล้าง จะเห็น banner แดงค้างคู่กับเขียวที่เพิ่งสำเร็จ
    setDeleteError("");
    setSuccessMsg(text);
    successTimerRef.current = setTimeout(() => setSuccessMsg(""), 4000);
  }, []);

  useEffect(() => {
    return () => {
      if (successTimerRef.current) clearTimeout(successTimerRef.current);
    };
  }, []);

  const handleDelete = async (p: Product) => {
    setDeleting(true);
    setDeleteError("");
    try {
      await adminProductService.delete(p.id);
      setConfirmDelete(null);
      list.refetch();
      flashSuccess(`ลบ "${p.name}" เรียบร้อย`);
    } catch (err) {
      // WHY: เดิมไม่มี catch — ลบไม่สำเร็จแล้วเงียบ ผู้ใช้เห็นสินค้ายังอยู่แล้วกดลบซ้ำ
      setConfirmDelete(null);
      setDeleteError(
        toUserMessage(err, `ลบ "${p.name}" ไม่สำเร็จ — อาจมีคำสั่งซื้อที่อ้างถึงสินค้านี้อยู่`)
      );
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-on-surface">จัดการสินค้า</h1>
          <p className="text-sm text-secondary">
            {list.hasActiveFilter
              ? `พบ ${list.total} รายการตามตัวกรอง`
              : `ทั้งหมด ${list.total} รายการ`}
          </p>
        </div>
        <button
          onClick={() => setAddOpen(true)}
          className="gradient-primary flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl active:scale-95"
        >
          <Plus size={18} />
          เพิ่มสินค้า
        </button>
      </div>

      <ProductFilters
        search={list.search}
        onSearchChange={list.setSearch}
        categories={list.flatCategories}
        categoryFilter={list.categoryFilter}
        onCategoryFilterChange={list.setCategoryFilter}
        statusFilter={list.statusFilter}
        onStatusFilterChange={list.setStatusFilter}
        hasActiveFilter={list.hasActiveFilter}
        onClearFilters={list.clearFilters}
      />

      {/* Error banner */}
      {list.fetchError && (
        <div className="mb-4 flex items-center gap-3 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600">
          <span className="font-semibold">เกิดข้อผิดพลาด</span>
          <span className="text-xs">
            ไม่สามารถโหลดข้อมูลสินค้าได้ — กรุณาตรวจสอบการเชื่อมต่อหรือลองใหม่
          </span>
          <button
            onClick={list.refetch}
            className="ml-auto rounded-lg bg-red-100 px-3 py-1 text-xs font-medium hover:bg-red-200"
          >
            ลองใหม่
          </button>
        </div>
      )}

      {/* Success banner */}
      {successMsg && (
        <div className="mb-4">
          <FormMessage ok text={successMsg} />
        </div>
      )}

      {/* Delete error banner */}
      {deleteError && (
        <div className="mb-4 flex items-center gap-3 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600">
          <AlertCircle size={16} className="shrink-0" />
          <span className="text-xs">{deleteError}</span>
          <button
            onClick={() => setDeleteError("")}
            aria-label="ปิด"
            className="ml-auto rounded-lg p-1 hover:bg-red-100"
          >
            <X size={14} />
          </button>
        </div>
      )}

      <div className="overflow-hidden rounded-2xl bg-white shadow-ambient">
        <ProductTable
          products={list.products}
          loading={list.loading}
          hasActiveFilter={list.hasActiveFilter}
          onClearFilters={list.clearFilters}
          onEdit={(product, tab) => setEditTarget({ product, tab })}
          onDelete={setConfirmDelete}
        />
        <PaginationBar
          page={list.page}
          totalPages={list.totalPages}
          total={list.total}
          limit={list.limit}
          pageButtons={list.pageButtons}
          onPageChange={list.setPage}
        />
      </div>

      {/* Modals */}
      <AddProductModal
        open={addOpen}
        onClose={() => setAddOpen(false)}
        categories={list.flatCategories}
        attributes={list.attributes}
        onCreated={(m) => {
          list.refetch();
          flashSuccess(m);
        }}
      />
      <ProductEditorModal
        open={!!editTarget}
        product={editTarget?.product ?? null}
        initialTab={editTarget?.tab ?? "general"}
        categories={list.flatCategories}
        attributes={list.attributes}
        onClose={() => setEditTarget(null)}
        onSaved={(m) => {
          list.refetch();
          setEditTarget(null);
          flashSuccess(m);
        }}
      />
      <ConfirmDialog
        open={!!confirmDelete}
        title={`ลบ "${confirmDelete?.name ?? ""}"?`}
        message={
          <>
            สินค้านี้มี {confirmDelete?.variants?.length ?? 0} variant ที่จะถูกลบไปด้วย
            <br />
            การลบไม่สามารถย้อนกลับได้
          </>
        }
        confirmLabel={deleting ? "กำลังลบ..." : "ลบสินค้า"}
        danger
        onConfirm={() => confirmDelete && handleDelete(confirmDelete)}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}
