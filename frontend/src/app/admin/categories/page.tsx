"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Plus, Pencil, Trash2, X, Check, Tag } from "lucide-react";
import { cn } from "@/lib/utils";
import { categoryService, adminCategoryService } from "@/lib/services";
import type { Category } from "@/lib/types";

/* ─── Create Category Modal ─── */
function CreateCategoryModal({
  open,
  onClose,
  categories,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  categories: Category[];
  onCreated: () => void;
}) {
  const [name, setName] = useState("");
  const [parentId, setParentId] = useState<number | "">("");
  const [saving, setSaving] = useState(false);

  const toSlug = (text: string) =>
    text.trim().toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9\u0E00-\u0E7F-]/g, "");

  const handleSubmit = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      await adminCategoryService.create({
        name: name.trim(),
        slug: toSlug(name),
        is_active: true,
        ...(parentId !== "" && { parent_id: parentId }),
      });
      onCreated();
      onClose();
      setName("");
      setParentId("");
    } finally {
      setSaving(false);
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/30 backdrop-blur-sm">
      <div className="relative w-full max-w-md rounded-2xl bg-white p-8 shadow-xl animate-in fade-in zoom-in-95 duration-200">
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-bold text-on-surface">เพิ่มหมวดหมู่</h2>
            <p className="text-xs text-secondary">กรอกชื่อหมวดหมู่ใหม่</p>
          </div>
          <button onClick={onClose} className="rounded-full p-1.5 text-secondary hover:bg-surface-highest hover:text-on-surface">
            <X size={20} />
          </button>
        </div>
        <div className="space-y-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-secondary">ชื่อหมวดหมู่ *</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="เช่น เสื้อผ้า"
              className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
              onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-secondary">หมวดหมู่แม่ (ถ้ามี)</label>
            <select
              value={parentId}
              onChange={(e) => setParentId(e.target.value ? Number(e.target.value) : "")}
              className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
            >
              <option value="">-- ไม่ระบุ (root) --</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="mt-6 flex justify-end gap-3">
          <button onClick={onClose} className="rounded-xl px-5 py-2.5 text-sm font-medium text-secondary hover:bg-surface-highest">
            ยกเลิก
          </button>
          <button
            onClick={handleSubmit}
            disabled={saving || !name.trim()}
            className="gradient-primary rounded-xl px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-primary/25 disabled:opacity-50"
          >
            {saving ? "กำลังบันทึก..." : "บันทึก"}
          </button>
        </div>
      </div>
    </div>
  );
}

/* ─── Category Row ─── */
function CategoryRow({
  cat,
  depth,
  onUpdated,
  flash,
}: {
  cat: Category;
  depth: number;
  onUpdated: () => void;
  flash: (ok: boolean, text: string) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [saving, setSaving] = useState(false);
  const [isActive, setIsActive] = useState(cat.is_active);

  useEffect(() => setIsActive(cat.is_active), [cat.is_active]);

  const startEdit = () => {
    setEditName(cat.name);
    setEditing(true);
  };

  const toSlug = (text: string) =>
    text.trim().toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9\u0E00-\u0E7F-]/g, "");

  const handleSaveEdit = async () => {
    if (!editName.trim()) return;
    setSaving(true);
    try {
      await adminCategoryService.update(cat.id, {
        name: editName.trim(),
        slug: toSlug(editName),
        is_active: isActive,
        ...(cat.parent_id !== null && { parent_id: cat.parent_id }),
      });
      flash(true, "แก้ไขชื่อสำเร็จ");
      setEditing(false);
      onUpdated();
    } catch (err) {
      flash(false, err instanceof Error ? err.message : "แก้ไขไม่สำเร็จ");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    const hasChildren = (cat.children?.length ?? 0) > 0;
    const msg = hasChildren
      ? `ต้องการลบ "${cat.name}" หรือไม่?\n\n⚠️ หมวดหมู่ย่อยทั้งหมด ${cat.children!.length} รายการภายใต้หมวดหมู่นี้จะถูกลบไปด้วย`
      : `ต้องการลบหมวดหมู่ "${cat.name}" หรือไม่?`;
    if (!confirm(msg)) return;
    try {
      await adminCategoryService.delete(cat.id);
      flash(true, "ลบหมวดหมู่สำเร็จ");
      onUpdated();
    } catch (err) {
      flash(false, err instanceof Error ? err.message : "ลบไม่สำเร็จ");
    }
  };

  const handleToggleActive = async () => {
    const next = !isActive;
    setIsActive(next);
    try {
      await adminCategoryService.toggleActive(cat.id, next);
    } catch (err) {
      setIsActive((prev) => !prev); // revert on error
      flash(false, err instanceof Error ? err.message : "เปลี่ยนสถานะไม่สำเร็จ");
    }
  };

  return (
    <tr className="border-b border-surface-highest/60 transition-colors hover:bg-surface-low/40">
      {/* Name */}
      <td className="px-6 py-4">
        {editing ? (
          <input
            type="text"
            value={editName}
            onChange={(e) => setEditName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleSaveEdit();
              if (e.key === "Escape") setEditing(false);
            }}
            autoFocus
            className="rounded-lg border border-primary/30 bg-surface-lowest px-3 py-1.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
          />
        ) : (
          <span style={{ paddingLeft: depth * 20 }} className="flex items-center gap-2 font-medium text-on-surface">
            {depth > 0 && <span className="text-outline">└</span>}
            {cat.name}
          </span>
        )}
      </td>
      {/* Slug */}
      <td className="px-4 py-4 font-mono text-xs text-secondary">{cat.slug}</td>
      {/* Status */}
      <td className="px-4 py-4">
        <button
          onClick={handleToggleActive}
          className={cn(
            "rounded-full px-2.5 py-0.5 text-[10px] font-semibold uppercase transition-colors hover:opacity-80",
            isActive ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-500"
          )}
        >
          {isActive ? "Active" : "Inactive"}
        </button>
      </td>
      {/* Actions */}
      <td className="px-4 py-4 text-right">
        <div className="flex items-center justify-end gap-1">
          {editing ? (
            <>
              <button
                onClick={handleSaveEdit}
                disabled={saving}
                className="rounded-lg p-2 text-secondary transition-colors hover:bg-emerald-50 hover:text-emerald-700"
                title="บันทึก"
              >
                <Check size={15} />
              </button>
              <button
                onClick={() => setEditing(false)}
                className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest"
                title="ยกเลิก"
              >
                <X size={15} />
              </button>
            </>
          ) : (
            <button
              onClick={startEdit}
              className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest hover:text-primary"
              title="แก้ไขชื่อ"
            >
              <Pencil size={15} />
            </button>
          )}
          <button
            onClick={handleDelete}
            className="rounded-lg p-2 text-secondary transition-colors hover:bg-red-50 hover:text-error"
            title="ลบหมวดหมู่"
          >
            <Trash2 size={15} />
          </button>
        </div>
      </td>
    </tr>
  );
}

/* ─── Categories Page ─── */
export default function AdminCategoriesPage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const flashTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchCategories = useCallback(async () => {
    setLoading(true);
    try {
      const res = await categoryService.list();
      setCategories(res ?? []);
    } catch {
      setCategories([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchCategories(); }, [fetchCategories]);

  const flash = useCallback((ok: boolean, text: string) => {
    if (flashTimerRef.current) clearTimeout(flashTimerRef.current);
    setMsg({ ok, text });
    flashTimerRef.current = setTimeout(() => setMsg(null), 3000);
  }, []);

  const flat = useMemo(() => {
    const result: { cat: Category; depth: number }[] = [];
    const flatten = (cats: Category[], depth: number) => {
      cats.forEach((c) => {
        result.push({ cat: c, depth });
        if (c.children?.length) flatten(c.children, depth + 1);
      });
    };
    flatten(categories, 0);
    return result;
  }, [categories]);

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-on-surface">จัดการหมวดหมู่</h1>
          <p className="text-sm text-secondary">ทั้งหมด {flat.length} รายการ</p>
        </div>
        <button
          onClick={() => setModalOpen(true)}
          className="gradient-primary flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl active:scale-95"
        >
          <Plus size={18} />
          เพิ่มหมวดหมู่
        </button>
      </div>

      {msg && (
        <div className={cn("mb-4 rounded-xl px-4 py-3 text-sm font-medium", msg.ok ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-600")}>
          {msg.text}
        </div>
      )}

      {/* Table */}
      <div className="overflow-hidden rounded-2xl bg-white shadow-ambient">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-surface-highest bg-surface-low/30 text-xs uppercase tracking-wider text-secondary">
              <th className="px-6 py-4 font-medium">ชื่อหมวดหมู่</th>
              <th className="px-4 py-4 font-medium">Slug</th>
              <th className="px-4 py-4 font-medium">สถานะ</th>
              <th className="px-4 py-4 font-medium text-right">จัดการ</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={4} className="py-16 text-center text-secondary">กำลังโหลด...</td>
              </tr>
            ) : flat.length === 0 ? (
              <tr>
                <td colSpan={4} className="py-16 text-center text-secondary">
                  <Tag size={40} className="mx-auto mb-2 text-outline" />
                  ยังไม่มีหมวดหมู่
                </td>
              </tr>
            ) : (
              flat.map(({ cat, depth }) => (
                <CategoryRow
                  key={cat.id}
                  cat={cat}
                  depth={depth}
                  onUpdated={fetchCategories}
                  flash={flash}
                />
              ))
            )}
          </tbody>
        </table>
      </div>

      <CreateCategoryModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        categories={categories}
        onCreated={fetchCategories}
      />
    </div>
  );
}
