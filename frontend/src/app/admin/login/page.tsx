"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ShieldCheck, User, Lock, ArrowRight } from "lucide-react";
import { adminAuthService } from "@/lib/services";
import { setAccessToken } from "@/lib/api";
import { useAuth } from "@/context/auth-context";

export default function AdminLoginPage() {
  const router = useRouter();
  const { refreshUser } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await adminAuthService.login({
        username,
        password,
        device_info: navigator.userAgent,
      });

      // What: เก็บ access_token ใน memory — refresh_token ถูก set เป็น HttpOnly cookie เรียบร้อยแล้ว
      setAccessToken(res.access_token);
      // What: refresh user state ใน AuthContext ก่อน navigate
      // Why:  AuthProvider fetchUser() ทำงานตอน mount แล้ว (ได้ null เพราะยังไม่มี token)
      //       ต้อง re-fetch เพื่อให้ layout เห็น user.role === "admin" ก่อน redirect
      await refreshUser();
      router.push("/admin");
    } catch (err) {
      setError(err instanceof Error ? err.message : "เกิดข้อผิดพลาด กรุณาลองอีกครั้ง");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-surface p-4">
      {/* Decorative bg */}
      <div className="pointer-events-none fixed inset-0 overflow-hidden">
        <div className="absolute -left-40 -top-40 h-96 w-96 rounded-full bg-primary/5 blur-3xl" />
        <div className="absolute -bottom-40 -right-40 h-96 w-96 rounded-full bg-secondary/5 blur-3xl" />
      </div>

      <div className="relative w-full max-w-md">
        {/* Card */}
        <div className="rounded-3xl bg-white p-10 shadow-ambient">
          {/* Icon */}
          <div className="mb-8 flex flex-col items-center">
            <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary-container/40 shadow-sm">
              <ShieldCheck size={28} className="text-primary" />
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-on-surface">
              เข้าสู่ระบบผู้ดูแล
            </h1>
            <p className="mt-1 text-sm text-secondary">
              Ananta Admin Console
            </p>
          </div>

          {/* Error */}
          {error && (
            <div className="mb-6 rounded-xl bg-error-container/20 px-4 py-3 text-sm text-error">
              {error}
            </div>
          )}

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label className="mb-1.5 block text-xs font-medium text-secondary">
                ชื่อผู้ใช้
              </label>
              <div className="group relative">
                <User
                  size={18}
                  className="absolute left-3.5 top-1/2 -translate-y-1/2 text-outline transition-colors group-focus-within:text-primary"
                />
                <input
                  type="text"
                  required
                  placeholder="admin"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full rounded-xl bg-surface-low/40 py-3 pl-11 pr-4 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20"
                />
              </div>
            </div>

            <div>
              <label className="mb-1.5 block text-xs font-medium text-secondary">
                รหัสผ่าน
              </label>
              <div className="group relative">
                <Lock
                  size={18}
                  className="absolute left-3.5 top-1/2 -translate-y-1/2 text-outline transition-colors group-focus-within:text-primary"
                />
                <input
                  type="password"
                  required
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded-xl bg-surface-low/40 py-3 pl-11 pr-4 text-sm outline-none transition-all focus:bg-surface-lowest focus:ring-2 focus:ring-primary/20"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="gradient-primary flex w-full items-center justify-center gap-2 rounded-xl py-3 text-sm font-semibold text-white shadow-lg shadow-primary/25 transition-all hover:shadow-xl active:scale-[0.98] disabled:opacity-50"
            >
              {loading ? (
                "กำลังเข้าสู่ระบบ..."
              ) : (
                <>
                  เข้าสู่ระบบ
                  <ArrowRight size={16} />
                </>
              )}
            </button>
          </form>

          {/* Back link */}
          <div className="mt-6 text-center">
            <a
              href="/"
              className="text-xs text-secondary transition-colors hover:text-primary"
            >
              ← กลับหน้าร้านค้า
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
