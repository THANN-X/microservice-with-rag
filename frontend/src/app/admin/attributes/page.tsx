"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Plus, Pencil, Trash2, X, Check, Tag, ChevronDown, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { attributeService, adminAttributeService } from "@/lib/services";
import type { Attribute, AttributeValue } from "@/lib/types";

/* ─── Attribute Row ─── */
function AttributeRow({
  attr,
  onUpdated,
  flash,
}: {
  attr: Attribute;
  onUpdated: () => void;
  flash: (ok: boolean, text: string) => void;
}) {
  const [values, setValues] = useState<AttributeValue[]>(attr.values ?? []);
  const [isExpanded, setIsExpanded] = useState(false);
  const [editingName, setEditingName] = useState(false);
  const [editAttrName, setEditAttrName] = useState("");
  const [addingValue, setAddingValue] = useState(false);
  const [newValueName, setNewValueName] = useState("");
  const [saving, setSaving] = useState(false);

  // Sync local values when parent re-fetches (e.g., after attr rename)
  useEffect(() => {
    setValues(attr.values ?? []);
  }, [attr]);

  const startEdit = () => {
    setEditAttrName(attr.name);
    setEditingName(true);
  };

  // Rename attr → still calls onUpdated() so parent reflects new name
  const handleSaveAttr = async () => {
    if (!editAttrName.trim()) return;
    try {
      await adminAttributeService.update(attr.id, { name: editAttrName.trim() });
      flash(true, "แก้ไขชื่อสำเร็จ");
      setEditingName(false);
      onUpdated();
    } catch (err) {
      flash(false, err instanceof Error ? err.message : "แก้ไขไม่สำเร็จ");
    }
  };

  // Delete attr → still calls onUpdated() so parent removes this row
  const handleDeleteAttr = async () => {
    if (!confirm(`ต้องการลบ attribute "${attr.name}" และค่าทั้งหมดของมันหรือไม่?`)) return;
    try {
      await adminAttributeService.delete(attr.id);
      flash(true, "ลบ attribute สำเร็จ");
      onUpdated();
    } catch (err) {
      flash(false, err instanceof Error ? err.message : "ลบไม่สำเร็จ");
    }
  };

  // Add value → append API response to local state, no re-fetch
  const handleAddValue = async () => {
    if (!newValueName.trim()) return;
    setSaving(true);
    try {
      const newVal = await adminAttributeService.createValue(attr.id, { value: newValueName.trim() });
      setValues((v) => [...v, newVal]);
      flash(true, `เพิ่มค่า "${newVal.value}" สำเร็จ`);
      setNewValueName("");
      setAddingValue(false);
    } catch (err) {
      flash(false, err instanceof Error ? err.message : "เพิ่มค่าไม่สำเร็จ");
    } finally {
      setSaving(false);
    }
  };

  // Delete value → filter local state, no re-fetch
  const handleDeleteValue = async (valueId: number, valueName: string) => {
    if (!confirm(`ต้องการลบค่า "${valueName}" หรือไม่?`)) return;
    try {
      await adminAttributeService.deleteValue(attr.id, valueId);
      setValues((v) => v.filter((val) => val.id !== valueId));
      flash(true, "ลบค่าสำเร็จ");
    } catch (err) {
      flash(false, err instanceof Error ? err.message : "ลบค่าไม่สำเร็จ");
    }
  };

  return (
    <div className="overflow-hidden rounded-2xl bg-white shadow-ambient">
      {/* Attribute header */}
      <div className="flex items-center gap-3 px-5 py-4">
        <button
          onClick={() => setIsExpanded((v) => !v)}
          className="text-secondary transition-colors hover:text-on-surface"
        >
          {isExpanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
        </button>

        {editingName ? (
          <div className="flex flex-1 items-center gap-2">
            <input
              type="text"
              value={editAttrName}
              onChange={(e) => setEditAttrName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleSaveAttr();
                if (e.key === "Escape") setEditingName(false);
              }}
              autoFocus
              className="flex-1 rounded-lg bg-surface-low/40 px-3 py-1.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
            />
            <button
              onClick={handleSaveAttr}
              className="rounded-lg p-1.5 text-emerald-600 hover:bg-emerald-50"
            >
              <Check size={16} />
            </button>
            <button
              onClick={() => setEditingName(false)}
              className="rounded-lg p-1.5 text-secondary hover:bg-surface-highest"
            >
              <X size={16} />
            </button>
          </div>
        ) : (
          <>
            <div className="flex-1">
              <p className="font-semibold text-on-surface">{attr.name}</p>
              <p className="text-xs text-secondary">
                {values.length} ค่า
                {values.length > 0 && (
                  <span className="ml-1 text-outline">
                    ({values.map((v) => v.value).slice(0, 5).join(", ")}
                    {values.length > 5 ? "..." : ""})
                  </span>
                )}
              </p>
            </div>
            <div className="flex items-center gap-1">
              <button
                onClick={() => { setAddingValue(true); if (!isExpanded) setIsExpanded(true); }}
                className="flex items-center gap-1 rounded-lg px-3 py-1.5 text-xs font-medium text-primary hover:bg-primary/10"
              >
                <Plus size={14} />
                เพิ่มค่า
              </button>
              <button
                onClick={startEdit}
                className="rounded-lg p-2 text-secondary hover:bg-surface-highest hover:text-on-surface"
                title="แก้ไขชื่อ"
              >
                <Pencil size={15} />
              </button>
              <button
                onClick={handleDeleteAttr}
                className="rounded-lg p-2 text-secondary hover:bg-red-50 hover:text-error"
                title="ลบ attribute"
              >
                <Trash2 size={15} />
              </button>
            </div>
          </>
        )}
      </div>

      {/* Expanded: values + add value form */}
      {isExpanded && (
        <div className="border-t border-surface-highest px-5 pb-4 pt-3">
          {addingValue && (
            <div className="mb-3 flex gap-2">
              <input
                type="text"
                placeholder={`เพิ่มค่าใหม่ใน "${attr.name}"`}
                value={newValueName}
                onChange={(e) => setNewValueName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleAddValue();
                  if (e.key === "Escape") { setAddingValue(false); setNewValueName(""); }
                }}
                autoFocus
                className="flex-1 rounded-xl bg-surface-low/40 px-4 py-2 text-sm outline-none focus:ring-2 focus:ring-primary/20"
              />
              <button
                onClick={handleAddValue}
                disabled={saving || !newValueName.trim()}
                className="gradient-primary rounded-xl px-4 py-2 text-xs font-semibold text-white disabled:opacity-50"
              >
                {saving ? "..." : "เพิ่ม"}
              </button>
              <button
                onClick={() => { setAddingValue(false); setNewValueName(""); }}
                className="rounded-xl p-2 text-secondary hover:bg-surface-highest"
              >
                <X size={16} />
              </button>
            </div>
          )}

          {values.length === 0 ? (
            <p className="text-xs text-outline">ยังไม่มีค่า — กด "เพิ่มค่า" ด้านบน</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {values.map((val) => (
                <div
                  key={val.id}
                  className="group flex items-center gap-1 rounded-lg border border-surface-highest bg-surface-low/30 px-3 py-1.5"
                >
                  <span className="text-sm text-on-surface">{val.value}</span>
                  <button
                    onClick={() => handleDeleteValue(val.id, val.value)}
                    className="ml-1 rounded p-0.5 text-outline opacity-0 transition-opacity hover:text-error group-hover:opacity-100"
                  >
                    <X size={13} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

/* ─── Attributes Page ─── */
export default function AdminAttributesPage() {
  const [attributes, setAttributes] = useState<Attribute[]>([]);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const flashTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [newAttrName, setNewAttrName] = useState("");
  const [creatingAttr, setCreatingAttr] = useState(false);

  const fetchAttributes = useCallback(async () => {
    setLoading(true);
    try {
      const res = await attributeService.list();
      setAttributes(res ?? []);
    } catch {
      setAttributes([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchAttributes(); }, [fetchAttributes]);

  const flash = useCallback((ok: boolean, text: string) => {
    if (flashTimerRef.current) clearTimeout(flashTimerRef.current);
    setMsg({ ok, text });
    flashTimerRef.current = setTimeout(() => setMsg(null), 3000);
  }, []);

  const handleCreateAttr = async () => {
    if (!newAttrName.trim()) return;
    setCreatingAttr(true);
    try {
      await adminAttributeService.create({ name: newAttrName.trim() });
      flash(true, `สร้าง attribute "${newAttrName.trim()}" สำเร็จ`);
      setNewAttrName("");
      await fetchAttributes();
    } catch {
      flash(false, "สร้าง attribute ไม่สำเร็จ");
    } finally {
      setCreatingAttr(false);
    }
  };

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-on-surface">จัดการ Attributes</h1>
          <p className="text-sm text-secondary">
            ทั้งหมด {attributes.length} รายการ · ใช้สำหรับกำหนดคุณสมบัติ variant เช่น สี, ขนาด
          </p>
        </div>
      </div>

      {msg && (
        <div className={cn("mb-4 rounded-xl px-4 py-3 text-sm font-medium", msg.ok ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-600")}>
          {msg.text}
        </div>
      )}

      {/* Create new attribute */}
      <div className="mb-6 rounded-2xl bg-white p-5 shadow-ambient">
        <h2 className="mb-3 text-sm font-semibold text-on-surface">เพิ่ม Attribute ใหม่</h2>
        <div className="flex gap-3">
          <input
            type="text"
            placeholder="เช่น สี, ขนาด, วัสดุ"
            value={newAttrName}
            onChange={(e) => setNewAttrName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleCreateAttr()}
            className="flex-1 rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20"
          />
          <button
            onClick={handleCreateAttr}
            disabled={creatingAttr || !newAttrName.trim()}
            className="gradient-primary flex items-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl disabled:opacity-50"
          >
            <Plus size={16} />
            {creatingAttr ? "กำลังสร้าง..." : "สร้าง"}
          </button>
        </div>
      </div>

      {/* Attribute list */}
      {loading ? (
        <div className="rounded-2xl bg-white p-12 text-center text-sm text-secondary shadow-ambient">
          กำลังโหลด...
        </div>
      ) : attributes.length === 0 ? (
        <div className="rounded-2xl bg-white p-12 text-center shadow-ambient">
          <Tag size={40} className="mx-auto mb-3 text-outline" />
          <p className="text-sm text-secondary">ยังไม่มี attribute — เริ่มสร้างด้านบน</p>
        </div>
      ) : (
        <div className="space-y-3">
          {attributes.map((attr) => (
            <AttributeRow
              key={attr.id}
              attr={attr}
              onUpdated={fetchAttributes}
              flash={flash}
            />
          ))}
        </div>
      )}
    </div>
  );
}
