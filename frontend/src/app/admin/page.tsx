// WHAT: แดชบอร์ดผู้ดูแล — ภาพรวมร้านค้า
// NOTE: การ์ด/กราฟในหน้านี้แสดงเฉพาะตัวเลขที่มาจาก API จริงเท่านั้น
//       ส่วนที่ backend ยังไม่มี endpoint ให้ (ยอดขายรายเดือน, สัดส่วนหมวดหมู่,
//       จำนวนลูกค้า, % เทียบเดือนก่อน) จะขึ้นสถานะ "ยังไม่มีข้อมูล" แทนการเดาตัวเลข
//       — dashboard ที่โชว์เลขปลอมอันตรายกว่า dashboard ที่โชว์น้อยแต่จริง
"use client";

import { useEffect, useState } from "react";
import {
  AlertCircle,
  BarChart3,
  DollarSign,
  Package,
  PieChart,
  ShoppingCart,
  TrendingDown,
  TrendingUp,
  Users,
} from "lucide-react";
import Link from "next/link";
import { cn, formatBaht } from "@/lib/utils";
import { adminOrderHistoryService, productService } from "@/lib/services";
import { adminOrderStatus } from "@/lib/order-status";
import type { AdminStats, OrderHistory } from "@/lib/types";

/* ─── Metric card ───
 * change = % เทียบช่วงก่อนหน้า จะแสดงก็ต่อเมื่อมีข้อมูลจริงเท่านั้น
 * note   = เหตุผลที่ยังไม่มีตัวเลขให้ดู (ใช้แทน change)
 */
function MetricCard({
  icon: Icon,
  label,
  value,
  accent,
  change,
  note,
}: {
  icon: typeof DollarSign;
  label: string;
  value: string;
  accent: string;
  change?: { pct: string; positive: boolean };
  note?: string;
}) {
  return (
    <div className="group relative overflow-hidden rounded-2xl bg-white p-6 shadow-ambient transition-all duration-300 hover:-translate-y-0.5 hover:shadow-lg">
      <div className="absolute -right-4 -top-4 opacity-5 transition-opacity group-hover:opacity-10">
        <Icon size={96} />
      </div>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-medium uppercase tracking-wider text-secondary">{label}</p>
          <p className="mt-2 text-3xl font-bold tracking-tight text-on-surface">{value}</p>
          {change ? (
            <div className="mt-2 flex items-center gap-1">
              {change.positive ? (
                <TrendingUp size={14} className="text-primary" />
              ) : (
                <TrendingDown size={14} className="text-error" />
              )}
              <span
                className={cn(
                  "text-xs font-medium",
                  change.positive ? "text-primary" : "text-error"
                )}
              >
                {change.pct}
              </span>
              <span className="text-xs text-outline">จากเดือนก่อน</span>
            </div>
          ) : (
            <p className="mt-2 text-xs text-outline">{note ?? " "}</p>
          )}
        </div>
        <div className={cn("flex h-12 w-12 items-center justify-center rounded-xl", accent)}>
          <Icon size={22} className="text-white" />
        </div>
      </div>
    </div>
  );
}

/* ─── กรอบการ์ดกราฟ ─── */
function ChartCard({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-2xl bg-white p-6 shadow-ambient">
      <div className="mb-4">
        <h3 className="text-sm font-bold text-on-surface">{title}</h3>
        {subtitle && <p className="text-xs text-secondary">{subtitle}</p>}
      </div>
      {children}
    </div>
  );
}

/** สถานะ "ยังไม่มีข้อมูล" ของกราฟ — บอกให้ชัดว่าไม่ใช่ศูนย์ แต่คือยังไม่มี API */
function ChartPlaceholder({
  icon: Icon,
  text,
  height = "h-[220px]",
}: {
  icon: typeof BarChart3;
  text: string;
  height?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-2 rounded-xl bg-surface-low/30 px-6 text-center",
        height
      )}
    >
      <Icon size={28} className="text-outline" />
      <p className="text-xs text-secondary">{text}</p>
    </div>
  );
}

/* ─── Sales chart (SVG area) ───
 * รับข้อมูลจริงผ่าน prop — ถ้ายังไม่มีจะขึ้น placeholder แทน
 * (เดิม hardcode array [30,50,40,...] ไว้ในตัว component ทำให้ดูเหมือนมียอดขายจริง)
 */
function SalesChart({ series }: { series: { label: string; value: number }[] }) {
  if (series.length === 0) {
    return (
      <ChartCard title="ยอดขายรายเดือน">
        <ChartPlaceholder
          icon={BarChart3}
          text="ยังไม่มีข้อมูลยอดขายรายเดือน"
        />
      </ChartCard>
    );
  }

  const width = 600;
  const height = 200;
  const padding = 20;
  const maxVal = Math.max(...series.map((s) => s.value)) || 1;
  const points = series.map((s, i) => ({
    x: padding + (i / Math.max(series.length - 1, 1)) * (width - 2 * padding),
    y: height - padding - (s.value / maxVal) * (height - 2 * padding),
  }));
  const pathLine = points.map((p, i) => `${i === 0 ? "M" : "L"}${p.x},${p.y}`).join(" ");
  const pathArea = `${pathLine} L${points[points.length - 1].x},${height - padding} L${points[0].x},${height - padding} Z`;

  return (
    <ChartCard title="ยอดขายรายเดือน">
      <svg viewBox={`0 0 ${width} ${height}`} className="w-full" preserveAspectRatio="none">
        <defs>
          <linearGradient id="areaFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#00675f" stopOpacity="0.20" />
            <stop offset="100%" stopColor="#00675f" stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={pathArea} fill="url(#areaFill)" />
        <path
          d={pathLine}
          fill="none"
          stroke="#00675f"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        {points.map((p, i) => (
          <circle key={i} cx={p.x} cy={p.y} r="4" fill="white" stroke="#00675f" strokeWidth="2" />
        ))}
      </svg>
      <div className="mt-2 flex justify-between px-5">
        {series.map((s) => (
          <span key={s.label} className="text-[10px] text-outline">
            {s.label}
          </span>
        ))}
      </div>
    </ChartCard>
  );
}

/* ─── Category donut chart ───
 * เดิม hardcode 35/25/20/20 พร้อมชื่อหมวดที่อาจไม่มีอยู่จริงในระบบ
 * ข้อมูลจริงต้อง join order_history (เก็บแค่ variant_id) → product → category
 */
function CategoryDonut({ slices }: { slices: { label: string; pct: number; color: string }[] }) {
  if (slices.length === 0) {
    return (
      <ChartCard title="สัดส่วนหมวดหมู่">
        <ChartPlaceholder
          icon={PieChart}
          text="ยังไม่มีข้อมูลสัดส่วนหมวดหมู่"
        />
      </ChartCard>
    );
  }

  const radius = 60;
  const circumference = 2 * Math.PI * radius;

  // คำนวณ offset สะสมไว้ล่วงหน้า แทนการ mutate ตัวแปรระหว่าง .map() ตอนเรนเดอร์
  // — ถ้า React เรนเดอร์ซ้ำโดยไม่ได้รีเซ็ต offset วงโดนัทจะเพี้ยน
  const arcs = slices.reduce<{ label: string; color: string; dash: number; offset: number }[]>(
    (acc, cat) => {
      const dash = (cat.pct / 100) * circumference;
      const offset = acc.length ? acc[acc.length - 1].offset + acc[acc.length - 1].dash : 0;
      acc.push({ label: cat.label, color: cat.color, dash, offset });
      return acc;
    },
    []
  );

  return (
    <ChartCard title="สัดส่วนหมวดหมู่">
      <div className="flex items-center justify-center">
        <svg width="160" height="160" viewBox="0 0 160 160">
          {arcs.map((arc) => (
            <circle
              key={arc.label}
              cx="80"
              cy="80"
              r={radius}
              fill="none"
              stroke={arc.color}
              strokeWidth="24"
              strokeDasharray={`${arc.dash} ${circumference - arc.dash}`}
              strokeDashoffset={-arc.offset}
              strokeLinecap="round"
              className="transition-all duration-500"
            />
          ))}
        </svg>
      </div>
      <div className="mt-4 space-y-2">
        {slices.map((cat) => (
          <div key={cat.label} className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="h-3 w-3 rounded-full" style={{ backgroundColor: cat.color }} />
              <span className="text-xs text-secondary">{cat.label}</span>
            </div>
            <span className="text-xs font-bold text-on-surface">{cat.pct}%</span>
          </div>
        ))}
      </div>
    </ChartCard>
  );
}

export default function AdminDashboardPage() {
  const [recentOrders, setRecentOrders] = useState<OrderHistory[]>([]);
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [productCount, setProductCount] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);

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
        // WHY: เดิม catch แล้ว console.error เฉย ๆ — การ์ดค้างเป็น "—" โดยผู้ใช้ไม่รู้ว่าโหลดพัง
        //      แยกไม่ออกระหว่าง "ยอดขาย 0" กับ "ต่อ API ไม่ได้"
        console.error("[AdminDashboard] failed to load dashboard data:", err);
        setLoadFailed(true);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  /** ค่าที่ยังโหลดอยู่ / โหลดพัง ให้แสดงขีดแทน 0 เพื่อไม่ให้เข้าใจผิดว่าไม่มียอด */
  const metric = (v: string | null | undefined) => (loading ? "…" : (v ?? "—"));

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold tracking-tight text-on-surface">แดชบอร์ดผู้ดูแล</h1>
        <p className="text-sm text-secondary">ภาพรวมร้านค้า (ยอดสะสมทั้งหมด)</p>
      </div>

      {loadFailed && (
        <div className="mb-6 flex items-center gap-3 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600">
          <AlertCircle size={16} className="shrink-0" />
          <span className="text-xs">
            โหลดข้อมูลแดชบอร์ดไม่สำเร็จ — ตัวเลขที่แสดงอาจไม่ครบ กรุณารีเฟรชหน้าอีกครั้ง
          </span>
        </div>
      )}

      {/* ── Metric Cards ── */}
      <div className="mb-8 grid grid-cols-1 gap-6 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          icon={DollarSign}
          label="ยอดขายรวม"
          value={metric(stats ? formatBaht(stats.total_revenue) : null)}
          note="นับเฉพาะคำสั่งซื้อที่ชำระเงินแล้ว"
          accent="bg-primary"
        />
        <MetricCard
          icon={ShoppingCart}
          label="คำสั่งซื้อที่ชำระแล้ว"
          value={metric(stats ? stats.total_orders.toLocaleString() : null)}
          accent="bg-secondary"
        />
        <MetricCard
          icon={Users}
          label="ลูกค้า"
          value="—"
          note="ยังไม่มี API นับจำนวนผู้ใช้"
          accent="bg-[#95c8fe]"
        />
        <MetricCard
          icon={Package}
          label="สินค้า"
          value={metric(productCount === null ? null : productCount.toLocaleString())}
          accent="bg-[#70516e]"
        />
      </div>

      {/* ── Charts row ── */}
      <div className="mb-8 grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <SalesChart series={[]} />
        </div>
        <CategoryDonut slices={[]} />
      </div>

      {/* ── Recent orders ── */}
      <div className="rounded-2xl bg-white p-6 shadow-ambient">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-sm font-bold text-on-surface">คำสั่งซื้อล่าสุด</h3>
          <Link href="/admin/orders" className="text-xs font-medium text-primary hover:underline">
            ดูทั้งหมด →
          </Link>
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
                    {loadFailed ? "โหลดคำสั่งซื้อไม่สำเร็จ" : "ยังไม่มีคำสั่งซื้อ"}
                  </td>
                </tr>
              ) : (
                recentOrders.map((o) => {
                  const status = adminOrderStatus(o.status);
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
                          className={cn("rounded-full px-3 py-1 text-xs font-medium", status.cls)}
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
