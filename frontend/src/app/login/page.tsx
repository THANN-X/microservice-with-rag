"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/context/auth-context";
import { GoogleLogin } from "@react-oauth/google";

export default function LoginPage() {
  const router = useRouter();
  const { login, loginWithGoogle } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [googleLoading, setGoogleLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(email, password);
      router.push("/");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "เข้าสู่ระบบไม่สำเร็จ");
    } finally {
      setLoading(false);
    }
  };

  const handleGoogleSuccess = async (credential: string) => {
    setGoogleLoading(true);
    setError("");
    try {
      await loginWithGoogle(credential);
      router.push("/");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "เข้าสู่ระบบด้วย Google ไม่สำเร็จ");
    } finally {
      setGoogleLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col">
      {/* Minimal Header */}
      <header className="px-8 py-6 flex justify-between items-center">
        <Link href="/" className="text-2xl font-black text-primary tracking-tight">อนันตา</Link>
      </header>

      <main className="flex-grow flex items-center justify-center px-4 pt-4 pb-12 relative overflow-hidden">
        {/* Background blurs */}
        <div className="absolute top-[-10%] right-[-5%] w-[400px] h-[400px] bg-primary-container/20 rounded-full blur-[100px] -z-10" />
        <div className="absolute bottom-[-10%] left-[-5%] w-[400px] h-[400px] bg-secondary-container/20 rounded-full blur-[100px] -z-10" />

        <div className="w-full max-w-[1100px] grid md:grid-cols-2 bg-surface-container-lowest rounded-[2rem] overflow-hidden shadow-2xl shadow-primary/5">
          {/* Branding Side */}
          <div className="hidden md:flex flex-col justify-between p-12 editorial-gradient text-white relative overflow-hidden">
            <div className="z-10">
              <h1 className="text-4xl font-black leading-tight mb-6 tracking-tight">
                สัมผัสประสบการณ์<br />ช้อปปิ้งที่เหนือระดับ
              </h1>
              <p className="text-lg opacity-90 max-w-xs thai-line-height">
                ค้นหาสินค้าคุณภาพคัดสรรพิเศษเพื่อคุณ พร้อมข้อเสนอสุดเอ็กซ์คลูซีฟที่มีเฉพาะที่ อนันตา เท่านั้น
              </p>
            </div>
            <div className="mt-12 relative z-10">
              <div className="bg-white/10 backdrop-blur-md rounded-xl p-6 border border-white/10">
                <div className="flex items-center gap-4 mb-4">
                  <div className="w-12 h-12 rounded-full bg-white/20 flex items-center justify-center">
                    <span className="material-symbols-outlined">local_offer</span>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-widest opacity-70">ข้อเสนอพิเศษ</p>
                    <p className="font-bold">รับส่วนลดเพิ่ม ฿ 500</p>
                  </div>
                </div>
                <p className="text-sm opacity-80 thai-line-height">
                  สำหรับการสั่งซื้อครั้งแรกผ่านแอปพลิเคชันอนันตา เอ็ดดิทอเรียล
                </p>
              </div>
            </div>
            <div className="absolute bottom-[-20%] right-[-10%] w-64 h-64 bg-primary-container/30 rounded-full blur-3xl" />
          </div>

          {/* Form Side */}
          <div className="p-8 md:p-12 lg:p-16 flex flex-col justify-center">
            <div className="mb-10">
              <div className="flex gap-8 border-b border-surface-container-highest mb-8">
                <span className="pb-4 text-lg font-bold text-primary border-b-2 border-primary">เข้าสู่ระบบ</span>
                <Link href="/register" className="pb-4 text-lg font-medium text-on-surface-variant hover:text-primary transition-colors">
                  สมัครสมาชิก
                </Link>
              </div>
              <h2 className="text-2xl font-bold text-on-surface mb-2">ยินดีต้อนรับกลับมา</h2>
              <p className="text-on-surface-variant thai-line-height">กรุณากรอกข้อมูลเพื่อเข้าใช้งานบัญชีของคุณ</p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-6">
              {error && (
                <div className="rounded-xl bg-error-container/10 border border-error/20 px-4 py-3 text-sm text-error">
                  {error}
                </div>
              )}

              <div>
                <label className="block text-sm font-semibold mb-2 text-on-surface-variant">
                  อีเมล หรือ เบอร์โทรศัพท์
                </label>
                <input
                  type="text"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-4 focus:ring-2 focus:ring-primary/20 focus:bg-surface-container-lowest transition-all outline-none thai-line-height"
                  placeholder="example@email.com"
                />
              </div>

              <div className="relative">
                <div className="flex justify-between items-center mb-2">
                  <label className="block text-sm font-semibold text-on-surface-variant">รหัสผ่าน</label>
                  <span className="text-xs font-bold text-primary">ลืมรหัสผ่าน?</span>
                </div>
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-4 focus:ring-2 focus:ring-primary/20 focus:bg-surface-container-lowest transition-all outline-none"
                  placeholder="••••••••"
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full editorial-gradient text-white font-bold py-4 rounded-xl shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 active:scale-[0.98] transition-all text-lg disabled:opacity-50"
              >
                {loading ? "กำลังเข้าสู่ระบบ..." : "เข้าสู่ระบบ"}
              </button>
            </form>

            <div className="mt-10">
              <div className="relative flex items-center justify-center mb-8">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-surface-container-highest" />
                </div>
                <span className="relative px-4 bg-surface-container-lowest text-sm text-on-surface-variant">
                  หรือเข้าใช้งานด้วย
                </span>
              </div>
              <div className="grid gap-4">
                {googleLoading ? (
                  <div className="flex items-center justify-center py-3 text-sm text-on-surface-variant">
                    กำลังเข้าสู่ระบบด้วย Google...
                  </div>
                ) : (
                  <GoogleLogin
                    onSuccess={(res) => {
                      if (res.credential) handleGoogleSuccess(res.credential);
                    }}
                    onError={() => setError("เข้าสู่ระบบด้วย Google ไม่สำเร็จ")}
                    theme="outline"
                    size="large"
                    width="100%"
                    text="signin_with"
                  />
                )}
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="w-full bg-surface-variant mt-auto">
        <div className="flex flex-col md:flex-row justify-between items-center px-10 py-12 gap-8 max-w-7xl mx-auto">
          <div className="flex flex-col gap-2 items-center md:items-start">
            <div className="text-lg font-bold text-primary">อนันตา</div>
            <p className="text-slate-700 text-sm">© {new Date().getFullYear()} อนันตา เอ็ดดิทอเรียล. สงวนลิขสิทธิ์.</p>
          </div>
          <div className="flex gap-8">
            <span className="text-slate-700 text-sm hover:text-primary transition-colors cursor-pointer">ข้อกำหนด</span>
            <span className="text-slate-700 text-sm hover:text-primary transition-colors cursor-pointer">ความเป็นส่วนตัว</span>
            <span className="text-slate-700 text-sm hover:text-primary transition-colors cursor-pointer">คำถามที่พบบ่อย</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
