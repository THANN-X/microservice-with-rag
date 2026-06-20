"use client";

import { useEffect, useState } from "react";
import { orderHistoryService } from "@/lib/services";
import { useAuth } from "@/context/auth-context";
import type { OrderHistory } from "@/lib/types";
import { formatBaht } from "@/lib/utils";
import Link from "next/link";

const ORDERS_PAGE = 1;
const ORDERS_PAGE_SIZE = 20;

const statusMap: Record<string, { label: string; color: string }> = {
  PENDING:           { label: "รอดำเนินการ",    color: "bg-amber-100 text-amber-800" },
  AWAITING_PAYMENT:  { label: "รอชำระเงิน",     color: "bg-amber-100 text-amber-800" },
  PAID:              { label: "ชำระเงินแล้ว",   color: "bg-emerald-100 text-emerald-800" },
  CONFIRMED:         { label: "ยืนยันแล้ว",     color: "bg-blue-100 text-blue-800" },
  SHIPPED:           { label: "กำลังจัดส่ง",    color: "bg-secondary-container/30 text-on-secondary-container" },
  COMPLETED:         { label: "จัดส่งสำเร็จ",   color: "bg-tertiary-container/30 text-on-tertiary-container" },
  CANCELLED:         { label: "ยกเลิก",         color: "bg-error-container/30 text-on-error-container" },
};

export default function OrdersPage() {
  const { user, loading: authLoading } = useAuth();
  const [orders, setOrders] = useState<OrderHistory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchOrders = () => {
    setError(null);
    setLoading(true);
    orderHistoryService.list(ORDERS_PAGE, ORDERS_PAGE_SIZE)
      .then((res) => setOrders(res.items || []))
      .catch(() => setError("ไม่สามารถโหลดข้อมูลคำสั่งซื้อได้ กรุณาลองใหม่อีกครั้ง"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    if (user) {
      fetchOrders();
    } else {
      setLoading(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user]);

  if (authLoading) {
    return (
      <div className="animate-pulse space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="bg-surface-container-lowest rounded-2xl p-6">
            <div className="flex gap-5">
              <div className="w-16 h-16 bg-surface-container-low rounded-xl" />
              <div className="flex-1 space-y-2">
                <div className="h-4 w-20 bg-surface-container-low rounded" />
                <div className="h-5 w-40 bg-surface-container-low rounded" />
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-6">
        <span className="material-symbols-outlined text-7xl text-outline">lock</span>
        <h1 className="text-3xl font-black text-on-surface">กรุณาเข้าสู่ระบบ</h1>
        <Link href="/login" className="editorial-gradient text-white px-8 py-4 rounded-full font-bold">
          เข้าสู่ระบบ
        </Link>
      </div>
    );
  }

  return (
    <>
      <div className="flex justify-between items-end mb-8">
        <div>
          <h1 className="text-2xl font-black text-on-surface tracking-tight">ประวัติการสั่งซื้อ</h1>
          <p className="text-on-surface-variant text-sm mt-1">{orders.length} คำสั่งซื้อ</p>
        </div>
      </div>

      {loading ? (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="bg-surface-container-lowest rounded-2xl p-6 animate-pulse">
              <div className="flex gap-5">
                <div className="w-16 h-16 bg-surface-container-low rounded-xl" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 w-20 bg-surface-container-low rounded" />
                  <div className="h-5 w-40 bg-surface-container-low rounded" />
                  <div className="h-3 w-24 bg-surface-container-low rounded" />
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : error ? (
        <div className="flex flex-col items-center justify-center py-20 gap-5">
          <span className="material-symbols-outlined text-5xl text-error">error</span>
          <p className="text-on-surface font-bold">{error}</p>
          <button
            onClick={fetchOrders}
            className="editorial-gradient text-white px-6 py-3 rounded-xl font-bold text-sm"
          >
            ลองใหม่
          </button>
        </div>
      ) : orders.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 gap-5">
          <span className="material-symbols-outlined text-7xl text-outline">receipt_long</span>
          <h2 className="text-xl font-bold text-on-surface">ยังไม่มีคำสั่งซื้อ</h2>
          <Link href="/products" className="editorial-gradient text-white px-7 py-3.5 rounded-xl font-bold text-sm">
            เริ่มช้อปปิ้ง
          </Link>
        </div>
      ) : (
        <div className="space-y-3">
          {orders.map((order) => {
            const status = statusMap[order.status] || statusMap.PENDING;
            return (
              <Link
                key={order.order_id}
                href={`/orders/${order.order_id}`}
                className="bg-surface-container-lowest rounded-2xl p-6 flex flex-col md:flex-row md:items-center justify-between gap-5 hover:shadow-lg hover:shadow-primary/5 transition-all border border-outline-variant/10 block"
              >
                <div className="flex gap-5">
                  <div className="w-16 h-16 bg-surface-container-low rounded-xl shrink-0 flex items-center justify-center">
                    <span className="material-symbols-outlined text-2xl text-outline">receipt_long</span>
                  </div>
                  <div className="space-y-1">
                    <span className={`text-[11px] font-bold px-2.5 py-0.5 rounded-full ${status.color}`}>
                      {status.label}
                    </span>
                    <h4 className="font-bold text-on-surface">
                      คำสั่งซื้อ #{order.order_id.slice(0, 8)}
                    </h4>
                    <p className="text-xs text-on-surface-variant">
                      {order.items?.length || 0} รายการ &middot; {new Date(order.created_at).toLocaleDateString("th-TH", { day: "numeric", month: "short", year: "numeric" })}
                    </p>
                  </div>
                </div>
                <div className="flex flex-row md:flex-col items-center md:items-end justify-between md:justify-center gap-2">
                  <p className="text-xl font-black text-primary leading-none">
                    {formatBaht(order.total_amount)}
                  </p>
                  <span className="text-xs font-bold text-primary flex items-center gap-1">
                    ดูรายละเอียด
                    <span className="material-symbols-outlined text-sm">chevron_right</span>
                  </span>
                </div>
              </Link>
            );
          })}
        </div>
      )}
    </>
  );
}
