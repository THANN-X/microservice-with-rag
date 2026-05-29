"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, type ReactNode } from "react";
import {
  LayoutDashboard,
  Package,
  ShoppingCart,
  Users,
  Settings,
  Tag,
  Layers,
  Search,
  Bell,
  HelpCircle,
  LogOut,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/context/auth-context";

const NAV_ITEMS = [
  { href: "/admin", icon: LayoutDashboard, label: "Dashboard" },
  { href: "/admin/products", icon: Package, label: "Products" },
  { href: "/admin/categories", icon: Tag, label: "Categories" },
  { href: "/admin/attributes", icon: Layers, label: "Attributes" },
  { href: "/admin/orders", icon: ShoppingCart, label: "Orders" },
  { href: "/admin/users", icon: Users, label: "Users" },
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
          {/* Search */}
          <div className="flex flex-1 items-center max-w-sm">
            <div className="relative w-full group">
              <Search
                size={15}
                className="absolute left-3.5 top-1/2 -translate-y-1/2 text-outline transition-colors group-focus-within:text-primary"
              />
              <input
                type="text"
                placeholder="ค้นหาข้อมูล..."
                className="w-full rounded-xl border border-surface-highest bg-surface/50 py-2 pl-9 pr-4 text-sm outline-none transition-all placeholder:text-outline focus:border-primary/30 focus:bg-white focus:ring-2 focus:ring-primary/10"
              />
            </div>
          </div>

          {/* Right */}
          <div className="flex items-center gap-1">
            <button className="relative rounded-xl p-2 text-outline transition-colors hover:bg-surface hover:text-on-surface">
              <Bell size={18} />
              <span className="absolute right-1.5 top-1.5 h-1.5 w-1.5 rounded-full bg-error" />
            </button>
            <button className="rounded-xl p-2 text-outline transition-colors hover:bg-surface hover:text-on-surface">
              <HelpCircle size={18} />
            </button>
            <div className="mx-3 h-6 w-px bg-surface-highest" />
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
