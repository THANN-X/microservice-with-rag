"use client";

import { useState } from "react";
import { Shield, UserPlus, Eye, EyeOff, CheckCircle } from "lucide-react";
import { adminAuthService } from "@/lib/services";
import type { CreateAdminRequest } from "@/lib/types";

const EMPTY_FORM: CreateAdminRequest & { admin_secret: string } = {
  first_name: "",
  last_name: "",
  username: "",
  password: "",
  phone: "",
  address: "",
  admin_secret: "",
};

export default function AdminsPage() {
  const [form, setForm] = useState(EMPTY_FORM);
  const [showPassword, setShowPassword] = useState(false);
  const [showSecret, setShowSecret] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

  const set = (field: string, value: string) =>
    setForm((f) => ({ ...f, [field]: value }));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSuccess(false);

    const { admin_secret, ...data } = form;
    if (!admin_secret) {
      setError("กรุณาใส่ Admin Secret Key");
      return;
    }

    setLoading(true);
    try {
      await adminAuthService.register(data, admin_secret);
      setSuccess(true);
      setForm(EMPTY_FORM);
    } catch (err) {
      setError(err instanceof Error ? err.message : "เกิดข้อผิดพลาด กรุณาลองใหม่");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto max-w-xl space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary-container/40">
          <Shield size={20} className="text-primary" />
        </div>
        <div>
          <h1 className="text-xl font-bold text-on-surface">สร้างผู้ดูแลระบบ</h1>
          <p className="text-sm text-secondary">เพิ่ม Admin Account ใหม่เข้าระบบ</p>
        </div>
      </div>

      {/* Success */}
      {success && (
        <div className="flex items-center gap-3 rounded-2xl bg-emerald-50 px-5 py-4 text-emerald-700">
          <CheckCircle size={18} />
          <span className="text-sm font-medium">สร้าง Admin สำเร็จแล้ว</span>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="rounded-2xl bg-error-container/20 px-5 py-4 text-sm text-error">
          {error}
        </div>
      )}

      {/* Form */}
      <form
        onSubmit={handleSubmit}
        className="rounded-3xl bg-white p-8 shadow-ambient space-y-5"
      >
        {/* Name row */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-secondary">
              ชื่อ <span className="text-error">*</span>
            </label>
            <input
              type="text"
              required
              minLength={2}
              placeholder="ชื่อจริง"
              value={form.first_name}
              onChange={(e) => set("first_name", e.target.value)}
              className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-secondary">
              นามสกุล <span className="text-error">*</span>
            </label>
            <input
              type="text"
              required
              minLength={2}
              placeholder="นามสกุล"
              value={form.last_name}
              onChange={(e) => set("last_name", e.target.value)}
              className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
            />
          </div>
        </div>

        {/* Username */}
        <div>
          <label className="mb-1 block text-xs font-medium text-secondary">
            Username <span className="text-error">*</span>
          </label>
          <input
            type="text"
            required
            placeholder="ชื่อผู้ใช้สำหรับ login"
            value={form.username}
            onChange={(e) => set("username", e.target.value)}
            className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
          />
        </div>

        {/* Password */}
        <div>
          <label className="mb-1 block text-xs font-medium text-secondary">
            Password <span className="text-error">*</span>
          </label>
          <div className="relative">
            <input
              type={showPassword ? "text" : "password"}
              required
              minLength={8}
              placeholder="อย่างน้อย 8 ตัวอักษร"
              value={form.password}
              onChange={(e) => set("password", e.target.value)}
              className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 pr-10 text-sm outline-none focus:ring-2 focus:ring-primary/20"
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-outline hover:text-on-surface"
            >
              {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>
        </div>

        {/* Phone */}
        <div>
          <label className="mb-1 block text-xs font-medium text-secondary">
            เบอร์โทรศัพท์ <span className="text-error">*</span>
          </label>
          <input
            type="tel"
            required
            minLength={10}
            placeholder="0812345678"
            value={form.phone}
            onChange={(e) => set("phone", e.target.value)}
            className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20"
          />
        </div>

        {/* Address */}
        <div>
          <label className="mb-1 block text-xs font-medium text-secondary">
            ที่อยู่ <span className="text-error">*</span>
          </label>
          <textarea
            required
            minLength={10}
            rows={2}
            placeholder="ที่อยู่"
            value={form.address}
            onChange={(e) => set("address", e.target.value)}
            className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20 resize-none"
          />
        </div>

        {/* Divider */}
        <hr className="border-outline-variant/30" />

        {/* Admin Secret */}
        <div>
          <label className="mb-1 block text-xs font-medium text-secondary">
            Admin Secret Key <span className="text-error">*</span>
          </label>
          <div className="relative">
            <input
              type={showSecret ? "text" : "password"}
              required
              placeholder="ADMIN_SECRET_KEY จาก environment"
              value={form.admin_secret}
              onChange={(e) => set("admin_secret", e.target.value)}
              className="w-full rounded-xl bg-amber-50 px-4 py-2.5 pr-10 text-sm outline-none focus:ring-2 focus:ring-amber-400/30"
            />
            <button
              type="button"
              onClick={() => setShowSecret((v) => !v)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-outline hover:text-on-surface"
            >
              {showSecret ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>
          <p className="mt-1.5 text-[11px] text-secondary">
            ต้องตรงกับค่า ADMIN_SECRET_KEY ใน env ของ auth service
          </p>
        </div>

        {/* Submit */}
        <button
          type="submit"
          disabled={loading}
          className="gradient-primary flex w-full items-center justify-center gap-2 rounded-xl py-3 text-sm font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl disabled:opacity-50"
        >
          <UserPlus size={16} />
          {loading ? "กำลังสร้าง..." : "สร้าง Admin Account"}
        </button>
      </form>
    </div>
  );
}
