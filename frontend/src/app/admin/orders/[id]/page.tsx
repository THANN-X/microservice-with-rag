"use client";

import { useEffect, useState, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  ArrowLeft,
  MapPin,
  Package,
  User,
  XCircle,
  Calendar,
  Hash,
} from "lucide-react";
import { cn, formatBaht } from "@/lib/utils";
import { adminOrderHistoryService, adminOrderService } from "@/lib/services";
import type { OrderHistory } from "@/lib/types";

const STATUS_MAP: Record<string, { label: string; cls: string }> = {
  PENDING: { label: "รอดำเนินการ", cls: "bg-amber-50 text-amber-700" },
  CONFIRMED: { label: "ยืนยันแล้ว", cls: "bg-blue-50 text-blue-700" },
  SHIPPED: { label: "จัดส่งแล้ว", cls: "bg-sky-50 text-sky-700" },
  COMPLETED: { label: "สำเร็จ", cls: "bg-emerald-50 text-emerald-700" },
  CANCELLED: { label: "ยกเลิก", cls: "bg-red-50 text-red-700" },
};

export default function AdminOrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();

  const [order, setOrder] = useState<OrderHistory | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  const [cancelReason, setCancelReason] = useState("");
  const [showCancel, setShowCancel] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const msgTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!id) return;
    adminOrderHistoryService
      .get(id)
      .then(setOrder)
      .catch(() => setNotFound(true))
      .finally(() => setLoading(false));
  }, [id]);

  const flash = (ok: boolean, text: string) => {
    if (msgTimerRef.current) clearTimeout(msgTimerRef.current);
    setMsg({ ok, text });
    msgTimerRef.current = setTimeout(() => setMsg(null), 3000);
  };

  const handleCancel = async () => {
    if (!order || !cancelReason.trim()) return;
    setCancelling(true);
    try {
      await adminOrderService.cancel(order.order_id, cancelReason.trim());
      setOrder((prev) => (prev ? { ...prev, status: "CANCELLED", cancel_reason: cancelReason.trim() } : prev));
      setShowCancel(false);
      setCancelReason("");
      flash(true, "ยกเลิกคำสั่งซื้อสำเร็จ");
    } catch {
      flash(false, "ไม่สามารถยกเลิกได้ กรุณาลองใหม่");
    } finally {
      setCancelling(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24 text-secondary">
        กำลังโหลด...
      </div>
    );
  }

  if (notFound || !order) {
    return (
      <div className="flex flex-col items-center justify-center py-24 gap-4">
        <Package size={48} className="text-outline" />
        <h2 className="text-lg font-bold text-on-surface">ไม่พบคำสั่งซื้อ</h2>
        <Link href="/admin/orders" className="text-sm text-primary hover:underline">
          กลับไปหน้าคำสั่งซื้อ
        </Link>
      </div>
    );
  }

  const status = STATUS_MAP[order.status] ?? STATUS_MAP.PENDING;
  const canCancel = order.status !== "CANCELLED" && order.status !== "COMPLETED";

  return (
    <div>
      {/* Back + Header */}
      <div className="mb-6">
        <Link
          href="/admin/orders"
          className="mb-4 inline-flex items-center gap-1.5 text-sm text-secondary hover:text-on-surface transition-colors"
        >
          <ArrowLeft size={15} />
          คำสั่งซื้อทั้งหมด
        </Link>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-on-surface">
              รายละเอียดคำสั่งซื้อ
            </h1>
            <p className="mt-0.5 font-mono text-sm text-secondary">#{order.order_id}</p>
          </div>
          <span className={cn("rounded-full px-4 py-1.5 text-xs font-semibold", status.cls)}>
            {status.label}
          </span>
        </div>
      </div>

      {msg && (
        <div
          className={cn(
            "mb-5 rounded-xl px-4 py-3 text-sm font-medium",
            msg.ok ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-600"
          )}
        >
          {msg.text}
        </div>
      )}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
        {/* ── Left: Order items ── */}
        <div className="lg:col-span-2 space-y-5">
          {/* Meta info */}
          <div className="rounded-2xl bg-white p-6 shadow-ambient">
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
              <div className="flex items-start gap-3">
                <div className="mt-0.5 rounded-lg bg-primary-container/30 p-2 text-primary">
                  <Hash size={14} />
                </div>
                <div>
                  <p className="text-[10px] uppercase tracking-wider text-secondary">Order ID</p>
                  <p className="mt-0.5 font-mono text-xs font-medium text-on-surface">
                    {order.order_id.slice(0, 8)}
                  </p>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <div className="mt-0.5 rounded-lg bg-primary-container/30 p-2 text-primary">
                  <User size={14} />
                </div>
                <div>
                  <p className="text-[10px] uppercase tracking-wider text-secondary">Customer ID</p>
                  <p className="mt-0.5 text-xs font-medium text-on-surface">{order.customer_id}</p>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <div className="mt-0.5 rounded-lg bg-primary-container/30 p-2 text-primary">
                  <Calendar size={14} />
                </div>
                <div>
                  <p className="text-[10px] uppercase tracking-wider text-secondary">วันที่สั่ง</p>
                  <p className="mt-0.5 text-xs font-medium text-on-surface">
                    {new Date(order.created_at).toLocaleDateString("th-TH", {
                      day: "numeric",
                      month: "short",
                      year: "numeric",
                    })}
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Items */}
          <div className="overflow-hidden rounded-2xl bg-white shadow-ambient">
            <div className="border-b border-surface-highest px-6 py-4">
              <h2 className="text-sm font-semibold text-on-surface">
                รายการสินค้า ({order.items?.length ?? 0} รายการ)
              </h2>
            </div>
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-surface-highest bg-surface-low/30 text-xs uppercase tracking-wider text-secondary">
                  <th className="px-6 py-3 text-left font-medium">Variant ID</th>
                  <th className="px-4 py-3 text-right font-medium">ราคา/ชิ้น</th>
                  <th className="px-4 py-3 text-right font-medium">จำนวน</th>
                  <th className="px-6 py-3 text-right font-medium">รวม</th>
                </tr>
              </thead>
              <tbody>
                {(order.items ?? []).map((item, i) => (
                  <tr
                    key={i}
                    className="border-b border-surface-highest/60 last:border-0 hover:bg-surface-low/40"
                  >
                    <td className="px-6 py-3 font-mono text-xs text-on-surface">
                      #{item.variant_id}
                    </td>
                    <td className="px-4 py-3 text-right text-secondary">
                      {formatBaht(item.unit_price)}
                    </td>
                    <td className="px-4 py-3 text-right text-secondary">{item.quantity}</td>
                    <td className="px-6 py-3 text-right font-medium text-on-surface">
                      {formatBaht(item.subtotal)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Note */}
          {order.note && (
            <div className="rounded-2xl bg-white p-6 shadow-ambient">
              <h2 className="mb-2 text-sm font-semibold text-on-surface">หมายเหตุ</h2>
              <p className="text-sm text-secondary">{order.note}</p>
            </div>
          )}

          {/* Cancel reason */}
          {order.cancel_reason && (
            <div className="rounded-2xl bg-red-50 p-6">
              <h2 className="mb-2 text-sm font-semibold text-red-700">เหตุผลยกเลิก</h2>
              <p className="text-sm text-red-600">{order.cancel_reason}</p>
            </div>
          )}
        </div>

        {/* ── Right: Summary + Address + Cancel ── */}
        <div className="space-y-5">
          {/* Total */}
          <div className="rounded-2xl bg-white p-6 shadow-ambient">
            <h2 className="mb-4 text-sm font-semibold text-on-surface">สรุปยอด</h2>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between text-secondary">
                <span>รายการทั้งหมด</span>
                <span>{order.items?.length ?? 0} รายการ</span>
              </div>
              <div className="flex justify-between border-t border-surface-highest pt-3">
                <span className="font-semibold text-on-surface">ยอดรวมสุทธิ</span>
                <span className="text-base font-bold text-primary">
                  {formatBaht(order.total_amount)}
                </span>
              </div>
            </div>
          </div>

          {/* Shipping address */}
          {order.shipping_address && (
            <div className="rounded-2xl bg-white p-6 shadow-ambient">
              <div className="mb-4 flex items-center gap-2">
                <MapPin size={14} className="text-primary" />
                <h2 className="text-sm font-semibold text-on-surface">ที่อยู่จัดส่ง</h2>
              </div>
              <div className="space-y-1 text-sm text-secondary">
                <p className="font-medium text-on-surface">
                  {order.shipping_address.full_name}
                </p>
                <p>{order.shipping_address.phone}</p>
                <p>{order.shipping_address.address_line}</p>
                <p>
                  {order.shipping_address.sub_district},{" "}
                  {order.shipping_address.district},{" "}
                  {order.shipping_address.province}{" "}
                  {order.shipping_address.postal_code}
                </p>
              </div>
            </div>
          )}

          {/* Cancel */}
          {canCancel && (
            <div className="rounded-2xl bg-white p-6 shadow-ambient">
              <h2 className="mb-4 text-sm font-semibold text-on-surface">การจัดการ</h2>
              {!showCancel ? (
                <button
                  onClick={() => setShowCancel(true)}
                  className="flex w-full items-center justify-center gap-2 rounded-xl border border-red-200 py-2.5 text-sm font-medium text-red-600 transition-colors hover:bg-red-50"
                >
                  <XCircle size={15} />
                  ยกเลิกคำสั่งซื้อ
                </button>
              ) : (
                <div className="space-y-3">
                  <label className="text-xs font-medium text-secondary">เหตุผลในการยกเลิก *</label>
                  <input
                    type="text"
                    value={cancelReason}
                    onChange={(e) => setCancelReason(e.target.value)}
                    placeholder="เช่น ยกเลิกโดยผู้ดูแลระบบ"
                    className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-red-200"
                  />
                  <div className="flex gap-2">
                    <button
                      onClick={() => { setShowCancel(false); setCancelReason(""); }}
                      className="flex-1 rounded-xl border border-surface-highest py-2 text-sm text-secondary hover:bg-surface-highest"
                    >
                      ยกเลิก
                    </button>
                    <button
                      onClick={handleCancel}
                      disabled={cancelling || !cancelReason.trim()}
                      className="flex-1 rounded-xl bg-red-500 py-2 text-sm font-semibold text-white disabled:opacity-50 hover:bg-red-600"
                    >
                      {cancelling ? "กำลังยกเลิก..." : "ยืนยัน"}
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
