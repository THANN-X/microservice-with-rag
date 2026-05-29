"use client";

import { useEffect, useState } from "react";
import {
  DollarSign,
  ShoppingCart,
  Users,
  Package,
  TrendingUp,
  TrendingDown,
} from "lucide-react";
import { cn, formatBaht } from "@/lib/utils";
import { adminOrderHistoryService, productService } from "@/lib/services";
import type { AdminStats, OrderHistory, Product } from "@/lib/types";

/* ─── Metric card ─── */
function MetricCard({
  icon: Icon,
  label,
  value,
  change,
  positive,
  accent,
}: {
  icon: typeof DollarSign;
  label: string;
  value: string;
  change: string;
  positive: boolean;
  accent: string;
}) {
  return (
    <div className="group relative overflow-hidden rounded-2xl bg-white p-6 shadow-ambient transition-all duration-300 hover:-translate-y-0.5 hover:shadow-lg">
      <div className="absolute -right-4 -top-4 opacity-5 transition-opacity group-hover:opacity-10">
        <Icon size={96} />
      </div>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-medium uppercase tracking-wider text-secondary">
            {label}
          </p>
          <p className="mt-2 text-3xl font-bold tracking-tight text-on-surface">
            {value}
          </p>
          <div className="mt-2 flex items-center gap-1">
            {positive ? (
              <TrendingUp size={14} className="text-primary" />
            ) : (
              <TrendingDown size={14} className="text-error" />
            )}
            <span
              className={cn(
                "text-xs font-medium",
                positive ? "text-primary" : "text-error"
              )}
            >
              {change}
            </span>
            <span className="text-xs text-outline">จากเดือนก่อน</span>
          </div>
        </div>
        <div
          className={cn(
            "flex h-12 w-12 items-center justify-center rounded-xl",
            accent
          )}
        >
          <Icon size={22} className="text-white" />
        </div>
      </div>
    </div>
  );
}

/* ─── Sales chart (SVG area) ─── */
function SalesChart() {
  const data = [30, 50, 40, 60, 45, 75, 65, 85, 70, 90, 80, 95];
  const labels = [
    "ม.ค.",
    "ก.พ.",
    "มี.ค.",
    "เม.ย.",
    "พ.ค.",
    "มิ.ย.",
    "ก.ค.",
    "ส.ค.",
    "ก.ย.",
    "ต.ค.",
    "พ.ย.",
    "ธ.ค.",
  ];
  const width = 600;
  const height = 200;
  const padding = 20;

  const maxVal = Math.max(...data);
  const points = data.map((v, i) => ({
    x: padding + (i / (data.length - 1)) * (width - 2 * padding),
    y: height - padding - (v / maxVal) * (height - 2 * padding),
  }));
  const pathLine = points.map((p, i) => `${i === 0 ? "M" : "L"}${p.x},${p.y}`).join(" ");
  const pathArea = `${pathLine} L${points[points.length - 1].x},${height - padding} L${points[0].x},${height - padding} Z`;

  return (
    <div className="rounded-2xl bg-white p-6 shadow-ambient">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-bold text-on-surface">ยอดขายรายเดือน</h3>
          <p className="text-xs text-secondary">ภาพรวมรายได้ปี 2024</p>
        </div>
        <select className="rounded-lg bg-surface-highest px-3 py-1.5 text-xs font-medium text-secondary outline-none">
          <option>รายเดือน</option>
          <option>รายสัปดาห์</option>
        </select>
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} className="w-full" preserveAspectRatio="none">
        <defs>
          <linearGradient id="areaFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#00675f" stopOpacity="0.20" />
            <stop offset="100%" stopColor="#00675f" stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={pathArea} fill="url(#areaFill)" />
        <path d={pathLine} fill="none" stroke="#00675f" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
        {points.map((p, i) => (
          <circle key={i} cx={p.x} cy={p.y} r="4" fill="white" stroke="#00675f" strokeWidth="2" />
        ))}
      </svg>
      {/* Labels */}
      <div className="mt-2 flex justify-between px-5">
        {labels.map((l) => (
          <span key={l} className="text-[10px] text-outline">
            {l}
          </span>
        ))}
      </div>
    </div>
  );
}

/* ─── Category donut chart ─── */
function CategoryDonut() {
  const categories = [
    { label: "เสื้อผ้า", pct: 35, color: "#00675f" },
    { label: "อุปกรณ์", pct: 25, color: "#70516e" },
    { label: "อิเล็กทรอนิกส์", pct: 20, color: "#95c8fe" },
    { label: "อื่นๆ", pct: 20, color: "#b5e6e6" },
  ];
  const radius = 60;
  const circumference = 2 * Math.PI * radius;
  let offset = 0;

  return (
    <div className="rounded-2xl bg-white p-6 shadow-ambient">
      <h3 className="mb-4 text-sm font-bold text-on-surface">สัดส่วนหมวดหมู่</h3>
      <div className="flex items-center justify-center">
        <svg width="160" height="160" viewBox="0 0 160 160">
          {categories.map((cat) => {
            const dash = (cat.pct / 100) * circumference;
            const gap = circumference - dash;
            const o = offset;
            offset += dash;
            return (
              <circle
                key={cat.label}
                cx="80"
                cy="80"
                r={radius}
                fill="none"
                stroke={cat.color}
                strokeWidth="24"
                strokeDasharray={`${dash} ${gap}`}
                strokeDashoffset={-o}
                strokeLinecap="round"
                className="transition-all duration-500"
              />
            );
          })}
        </svg>
      </div>
      <div className="mt-4 space-y-2">
        {categories.map((cat) => (
          <div key={cat.label} className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span
                className="h-3 w-3 rounded-full"
                style={{ backgroundColor: cat.color }}
              />
              <span className="text-xs text-secondary">{cat.label}</span>
            </div>
            <span className="text-xs font-bold text-on-surface">{cat.pct}%</span>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ─── Status badge ─── */
const STATUS_MAP: Record<string, { label: string; cls: string }> = {
  PENDING: { label: "รอดำเนินการ", cls: "bg-amber-50 text-amber-700" },
  CONFIRMED: { label: "ยืนยันแล้ว", cls: "bg-blue-50 text-blue-700" },
  SHIPPED: { label: "จัดส่งแล้ว", cls: "bg-sky-50 text-sky-700" },
  COMPLETED: { label: "สำเร็จ", cls: "bg-emerald-50 text-emerald-700" },
  CANCELLED: { label: "ยกเลิก", cls: "bg-red-50 text-red-700" },
};

export default function AdminDashboardPage() {
  const [recentOrders, setRecentOrders] = useState<OrderHistory[]>([]);
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [productCount, setProductCount] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const [ordersRes, statsRes, productsRes] = await Promise.all([
          adminOrderHistoryService.list(1, 5),
          adminOrderHistoryService.stats(),
          productService.list({ page: 1, limit: 1 }),
        ]);
        setRecentOrders(ordersRes.items ?? []);
        setStats(statsRes);
        setProductCount(productsRes.total ?? 0);
      } catch (err) {
        console.error("[AdminDashboard] failed to load dashboard data:", err);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return (
    <div>
      <div className="mb-8 flex items-end justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-on-surface">
            แดชบอร์ดผู้ดูแล
          </h1>
          <p className="text-sm text-secondary">ภาพรวมร้านค้าและข้อมูลเชิงลึก</p>
        </div>
        <div className="flex gap-2">
          <select className="rounded-lg bg-white px-4 py-2 text-sm font-medium text-secondary shadow-ambient outline-none">
            <option>7 วันล่าสุด</option>
            <option>30 วันล่าสุด</option>
            <option>ปีนี้</option>
          </select>
        </div>
      </div>

      {/* ── Metric Cards ── */}
      <div className="mb-8 grid grid-cols-1 gap-6 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          icon={DollarSign}
          label="ยอดขายรวม"
          value={stats ? formatBaht(stats.total_revenue) : "—"}
          change="+12.5%"
          positive
          accent="bg-primary"
        />
        <MetricCard
          icon={ShoppingCart}
          label="คำสั่งซื้อ"
          value={stats ? stats.total_orders.toLocaleString() : "—"}
          change="+8.2%"
          positive
          accent="bg-secondary"
        />
        <MetricCard
          icon={Users}
          label="ลูกค้า"
          value="1,245"
          change="+5.1%"
          positive
          accent="bg-[#95c8fe]"
        />
        <MetricCard
          icon={Package}
          label="สินค้า"
          value={String(productCount)}
          change="-2.4%"
          positive={false}
          accent="bg-[#70516e]"
        />
      </div>

      {/* ── Charts row ── */}
      <div className="mb-8 grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <SalesChart />
        </div>
        <CategoryDonut />
      </div>

      {/* ── Recent orders ── */}
      <div className="rounded-2xl bg-white p-6 shadow-ambient">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-sm font-bold text-on-surface">คำสั่งซื้อล่าสุด</h3>
          <a
            href="/admin/orders"
            className="text-xs font-medium text-primary hover:underline"
          >
            ดูทั้งหมด →
          </a>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-surface-highest text-xs uppercase tracking-wider text-secondary">
                <th className="py-3 pr-4 font-medium">Order ID</th>
                <th className="py-3 pr-4 font-medium">วันที่</th>
                <th className="py-3 pr-4 font-medium">จำนวนเงิน</th>
                <th className="py-3 pr-4 font-medium">สถานะ</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={4} className="py-12 text-center text-secondary">
                    กำลังโหลด...
                  </td>
                </tr>
              ) : recentOrders.length === 0 ? (
                <tr>
                  <td colSpan={4} className="py-12 text-center text-secondary">
                    ยังไม่มีคำสั่งซื้อ
                  </td>
                </tr>
              ) : (
                recentOrders.map((o) => {
                  const status = STATUS_MAP[o.status] ?? STATUS_MAP.PENDING;
                  return (
                    <tr
                      key={o.order_id}
                      className="border-b border-surface-highest/60 transition-colors hover:bg-surface-low/40"
                    >
                      <td className="py-3 pr-4 font-medium text-on-surface">
                        #{o.order_id?.slice(0, 8)}
                      </td>
                      <td className="py-3 pr-4 text-secondary">
                        {new Date(o.created_at).toLocaleDateString("th-TH")}
                      </td>
                      <td className="py-3 pr-4 font-medium text-on-surface">
                        {formatBaht(o.total_amount)}
                      </td>
                      <td className="py-3 pr-4">
                        <span
                          className={cn(
                            "rounded-full px-3 py-1 text-xs font-medium",
                            status.cls
                          )}
                        >
                          {status.label}
                        </span>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
