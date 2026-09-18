"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, type ReactNode } from "react";
import {
  LayoutDashboard,
  Package,
  ShoppingCart,
  Settings,
  Tag,
  Layers,
  LogOut,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/context/auth-context";

// NOTE: เมนู "Users" (/admin/users) ถูกเอาออกชั่วคราว — ยังไม่มีไฟล์ page.tsx ของ route นี้
//       กดแล้วเจอ 404 ใส่กลับได้ทันทีเมื่อสร้างหน้าเสร็จ
const NAV_ITEMS = [
  { href: "/admin", icon: LayoutDashboard, label: "Dashboard" },
  { href: "/admin/products", icon: Package, label: "Products" },
  { href: "/admin/categories", icon: Tag, label: "Categories" },
  { href: "/admin/attributes", icon: Layers, label: "Attributes" },
  { href: "/admin/orders", icon: ShoppingCart, label: "Orders" },
  { href: "/admin/admins", icon: Settings, label: "Admins" },
];

export default function AdminLayout({ children }: { children: ReactNode }) {
  return (
    <Suspense fallback={null}>
      <AdminLayoutInner>{children}</AdminLayoutInner>
    </Suspense>
  );
}

function AdminLayoutInner({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const { user, loading, logout } = useAuth();

  const isLoginPage = pathname === "/admin/login";
  const isPreview = searchParams.get("preview") === "true";

  // What: ตรวจ role admin หลัง auth-context โหลด user เสร็จ
  // Why:  ป้องกัน FOUC — แสดง skeleton ขณะที่ยังตรวจสอบอยู่
  //       ไม่ใช้ localStorage หรือ atob decode เพราะ XSS อ่านได้ และ decode ไม่ verify signature
  useEffect(() => {
    if (isLoginPage || isPreview || loading) return;
    if (!user || user.role !== "admin") {
      router.push("/admin/login");
    }
  }, [user, loading, router, isLoginPage, isPreview]);

  const handleLogout = () => {
    logout();
    router.push("/admin/login");
  };

  const adminName = user ? `${user.first_name} ${user.last_name}`.trim() || "Admin" : "Admin";

  const isActive = (href: string) =>
    href === "/admin" ? pathname === "/admin" : pathname.startsWith(href);

  // ชื่อหน้าปัจจุบันสำหรับ breadcrumb — หา match ที่ยาวที่สุดเพื่อรองรับ nested route
  // เช่น /admin/orders/123 ต้องได้ "Orders" ไม่ใช่ "Dashboard"
  const currentPageLabel =
    NAV_ITEMS.filter((item) => isActive(item.href)).sort(
      (a, b) => b.href.length - a.href.length
    )[0]?.label ?? "Dashboard";

  /* Login page gets no chrome */
  if (isLoginPage) return <>{children}</>;

  /* Show skeleton while checking auth to prevent FOUC */
  if (loading || !user || user.role !== "admin") {
    return (
      <div className="flex min-h-screen items-center justify-center bg-surface">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
      </div>
    );
  }

  return (
    <div className="flex min-h-screen bg-surface">
      {/* ═══════ Sidebar ═══════ */}
      <aside className="fixed left-0 top-0 z-50 flex h-screen w-64 flex-col border-r-0 bg-surface py-6">
        {/* Brand */}
        <div className="mb-10 px-6">
          <span className="text-2xl font-bold tracking-tight text-primary">
            อนันตา
          </span>
          <p className="mt-1 text-[10px] font-medium uppercase tracking-widest text-secondary">
            Admin Console
          </p>
        </div>

        {/* Nav */}
        <nav className="flex-1 space-y-1">
          {NAV_ITEMS.map(({ href, icon: Icon, label }) => (
            <Link
              key={href}
              href={href}
              className={cn(
                "mx-2 flex items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium transition-all duration-200 active:scale-95",
                isActive(href)
                  ? "bg-surface-lowest text-primary shadow-sm"
                  : "text-secondary hover:bg-surface-low"
              )}
            >
              <Icon size={20} />
              <span>{label}</span>
            </Link>
          ))}
        </nav>

        {/* User */}
        <div className="mt-auto px-6">
          <div className="rounded-xl bg-surface-highest p-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-container font-bold text-primary">
                {adminName.charAt(0).toUpperCase()}
              </div>
              <div className="flex-1 min-w-0">
                <p className="truncate text-xs font-bold text-on-surface">
                  {adminName}
                </p>
                <p className="text-[10px] text-secondary">{user.username}</p>
              </div>
              <button
                onClick={handleLogout}
                className="rounded-full p-1.5 text-secondary transition-colors hover:bg-surface-low hover:text-primary"
                title="ออกจากระบบ"
              >
                <LogOut size={14} />
              </button>
            </div>
          </div>
        </div>
      </aside>

      {/* ═══════ Main area ═══════ */}
      <div className="ml-64 flex flex-1 flex-col">
        {/* Top Header */}
        <header className="sticky top-0 z-40 flex h-16 items-center justify-between border-b border-surface-highest/60 bg-white/80 px-8 backdrop-blur-md">
          {/* Breadcrumb
              WHY: ตรงนี้เคยเป็นช่องค้นหา + กระดิ่งแจ้งเตือน + ปุ่มช่วยเหลือ ที่กดแล้วไม่มีอะไรเกิดขึ้น
                   ช่องค้นหาซ้ำซ้อนกับช่องค้นหาในแต่ละหน้าจนผู้ใช้พิมพ์ผิดช่อง
                   ส่วนกระดิ่งมีจุดแดงค้างตลอดเหมือนมีของใหม่ตลอดเวลา
                   เอาออกก่อนจนกว่าจะมีของจริงมารองรับ แล้วใส่ breadcrumb ที่บอกตำแหน่งจริงแทน */}
          <div className="flex flex-1 items-center gap-2 text-sm">
            <span className="text-outline">Admin</span>
            <span className="text-outline">/</span>
            <span className="font-medium text-on-surface">{currentPageLabel}</span>
          </div>

          {/* Right */}
          <div className="flex items-center gap-1">
            <div className="flex items-center gap-2.5 rounded-xl border border-surface-highest bg-surface/60 px-3 py-1.5 transition-colors hover:bg-surface">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg editorial-gradient text-[11px] font-bold text-white shadow-sm">
                {adminName.charAt(0).toUpperCase()}
              </div>
              <div className="flex flex-col leading-none">
                <span className="text-xs font-semibold text-on-surface">{adminName}</span>
                <span className="text-[10px] text-outline">Administrator</span>
              </div>
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 p-8">{children}</main>
      </div>
    </div>
  );
}
