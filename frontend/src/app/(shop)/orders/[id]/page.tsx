"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { orderService } from "@/lib/services";
import { useAuth } from "@/context/auth-context";
import type { Order } from "@/lib/types";
import { formatBaht } from "@/lib/utils";
import Image from "next/image";
import Link from "next/link";

const statusMap: Record<string, { label: string; color: string }> = {
  PENDING:           { label: "รอดำเนินการ",    color: "bg-amber-100 text-amber-800" },
  AWAITING_PAYMENT:  { label: "รอชำระเงิน",     color: "bg-amber-100 text-amber-800" },
  PAID:              { label: "ชำระเงินแล้ว",   color: "bg-emerald-100 text-emerald-800" },
  CONFIRMED:         { label: "ยืนยันแล้ว",     color: "bg-blue-100 text-blue-800" },
  SHIPPED:           { label: "กำลังจัดส่ง",    color: "bg-secondary-container/30 text-on-secondary-container" },
  COMPLETED:         { label: "จัดส่งสำเร็จ",   color: "bg-tertiary-container/30 text-on-tertiary-container" },
  CANCELLED:         { label: "ยกเลิก",         color: "bg-error-container/30 text-on-error-container" },
};

export default function OrderDetailPage() {
  const { id } = useParams();
  const { user, loading: authLoading } = useAuth();
  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [cancelReason, setCancelReason] = useState("");
  const [showCancel, setShowCancel] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [cancelMsg, setCancelMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [repayQr, setRepayQr] = useState<string | null>(null);
  const [repayLoading, setRepayLoading] = useState(false);
  const [repayError, setRepayError] = useState<string | null>(null);

  useEffect(() => {
    if (id) {
      orderService.get(String(id))
        .then(setOrder)
        .catch(() => setError("ไม่สามารถโหลดข้อมูลคำสั่งซื้อได้ กรุณาลองใหม่อีกครั้ง"))
        .finally(() => setLoading(false));
    }
  }, [id]);

  const handleCancel = async () => {
    if (!order || !cancelReason.trim()) return;
    setCancelling(true);
    try {
      await orderService.cancel(order.id, cancelReason.trim());
      setOrder((prev) => prev ? { ...prev, status: "CANCELLED" } : prev);
      setShowCancel(false);
      setCancelReason("");
      setCancelMsg({ ok: true, text: "ยกเลิกคำสั่งซื้อสำเร็จ" });
    } catch {
      setCancelMsg({ ok: false, text: "ไม่สามารถยกเลิกได้" });
    } finally {
      setCancelling(false);
    }
  };

  const handleRepay = async () => {
    if (!order) return;
    setRepayLoading(true);
    setRepayQr(null);
    setRepayError(null);
    try {
      const res = await orderService.processPayment(order.id, {
        token: "",
        payment_method: "PROMPTPAY",
      });
      const qr = res.payment.qr_image_url;
      if (qr) {
        setRepayQr(qr);
        // order อาจเพิ่งเปลี่ยน CONFIRMED → AWAITING_PAYMENT หลัง generate QR
        setOrder((prev) => prev && prev.status === "CONFIRMED" ? { ...prev, status: "AWAITING_PAYMENT" } : prev);
      } else setRepayError("ไม่ได้รับ QR Code กรุณาลองใหม่");
    } catch {
      setRepayError("ไม่สามารถโหลด QR Code ได้ กรุณาลองใหม่");
    } finally {
      setRepayLoading(false);
    }
  };

  if (authLoading || loading) {
    return (
      <div className="flex flex-col items-center justify-center py-20 gap-3">
        <span className="material-symbols-outlined text-5xl text-primary animate-pulse">receipt_long</span>
        <p className="text-sm text-on-surface-variant">กำลังโหลด...</p>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-6">
        <span className="material-symbols-outlined text-7xl text-outline">lock</span>
        <h1 className="text-2xl font-black text-on-surface">กรุณาเข้าสู่ระบบ</h1>
        <Link href="/login" className="editorial-gradient text-white px-8 py-4 rounded-full font-bold">
          เข้าสู่ระบบ
        </Link>
      </div>
    );
  }

  if (!order) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-6">
        <span className="material-symbols-outlined text-7xl text-outline">{error ? "error" : "receipt_long"}</span>
        <h1 className="text-2xl font-black text-on-surface">{error ? "เกิดข้อผิดพลาด" : "ไม่พบคำสั่งซื้อ"}</h1>
        {error && <p className="text-sm text-on-surface-variant">{error}</p>}
        <Link href="/orders" className="text-primary font-bold hover:underline">กลับไปหน้าคำสั่งซื้อ</Link>
      </div>
    );
  }

  const status = statusMap[order.status] || statusMap.PENDING;

  return (
    <>
      <div className="flex items-center gap-2 text-sm text-on-surface-variant mb-6">
        <Link href="/orders" className="hover:text-primary transition-colors">คำสั่งซื้อ</Link>
        <span className="material-symbols-outlined text-xs">chevron_right</span>
        <span className="text-on-surface font-medium">#{order.id.slice(0, 8)}</span>
      </div>

      {cancelMsg && (
        <div className={`mb-6 rounded-xl px-5 py-3.5 text-sm font-medium flex items-center gap-2 ${
          cancelMsg.ok ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-600"
        }`}>
          <span className="material-symbols-outlined text-base" style={{ fontVariationSettings: "'FILL' 1" }}>
            {cancelMsg.ok ? "check_circle" : "error"}
          </span>
          {cancelMsg.text}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div className="lg:col-span-2 space-y-5">
          <div className="bg-surface-container-lowest rounded-2xl p-7 border border-outline-variant/10">
            <div className="flex justify-between items-start mb-5">
              <div>
                <h1 className="text-xl font-black text-on-surface">คำสั่งซื้อ #{order.id.slice(0, 8)}</h1>
                <p className="text-sm text-on-surface-variant mt-1">
                  วันที่: {new Date(order.created_at).toLocaleDateString("th-TH", { year: "numeric", month: "long", day: "numeric" })}
                </p>
              </div>
              <span className={`text-[11px] font-bold px-3 py-1 rounded-full ${status.color}`}>
                {status.label}
              </span>
            </div>

            <div className="space-y-0">
              {order.items?.map((item) => (
                <div key={item.id} className="flex gap-3.5 items-center py-4 border-b border-outline-variant/10 last:border-0">
                  {item.image_url && (
                    <div className="w-14 h-14 rounded-xl bg-surface-container-low flex-shrink-0 overflow-hidden relative">
                      <Image src={item.image_url} alt={item.product_name || ""} fill className="object-cover" sizes="56px" />
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="font-bold text-sm text-on-surface truncate">{item.product_name || `Variant #${item.variant_id}`}</p>
                    {item.variant_name && <p className="text-xs text-on-surface-variant mt-0.5">{item.variant_name}</p>}
                    <p className="text-xs text-on-surface-variant mt-0.5">จำนวน: {item.quantity}</p>
                  </div>
                  <p className="font-bold text-sm text-primary flex-shrink-0">{formatBaht(item.subtotal)}</p>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="space-y-5">
          <div className="bg-surface-container-lowest rounded-2xl p-7 border border-outline-variant/10">
            <h3 className="font-bold text-on-surface mb-4 text-sm">สรุปยอดชำระ</h3>
            <div className="space-y-3">
              <div className="flex justify-between text-sm">
                <span className="text-on-surface-variant">ยอดรวม</span>
                <span className="font-medium">{formatBaht(order.total_amount)}</span>
              </div>
              <div className="pt-4 border-t border-outline-variant/10 flex justify-between items-end">
                <span className="font-bold text-sm">ยอดชำระสุทธิ</span>
                <span className="text-xl font-black text-primary">{formatBaht(order.total_amount)}</span>
              </div>
            </div>
          </div>

          {order.shipping_address && (
            <div className="bg-surface-container-lowest rounded-2xl p-7 border border-outline-variant/10">
              <h3 className="font-bold text-on-surface mb-4 text-sm">ที่อยู่จัดส่ง</h3>
              <div className="text-sm text-on-surface-variant space-y-1">
                <p className="font-medium text-on-surface">{order.shipping_address.full_name}</p>
                <p>{order.shipping_address.phone}</p>
                <p>{order.shipping_address.address_line}</p>
                <p>{order.shipping_address.sub_district}, {order.shipping_address.district}, {order.shipping_address.province} {order.shipping_address.postal_code}</p>
              </div>
            </div>
          )}

          {/* Pay now section */}
          {(order.status === "AWAITING_PAYMENT" || order.status === "CONFIRMED") && (
            <div className="bg-surface-container-lowest rounded-2xl p-7 border border-primary/20">
              <div className="flex items-center gap-2 mb-1">
                <span className="material-symbols-outlined text-primary text-[18px]" style={{ fontVariationSettings: "'FILL' 1" }}>qr_code_2</span>
                <h3 className="font-bold text-on-surface text-sm">ชำระเงิน</h3>
              </div>
              <p className="text-xs text-on-surface-variant mb-4">คำสั่งซื้อนี้รอการชำระเงิน</p>
              {repayQr ? (
                <div className="flex flex-col items-center gap-3">
                  <div className="bg-white p-3 rounded-xl border border-outline-variant/20">
                    <Image src={repayQr} alt="PromptPay QR Code" width={200} height={200} />
                  </div>
                  <p className="text-xs text-on-surface-variant text-center">สแกน QR เพื่อชำระเงิน</p>
                </div>
              ) : (
                <>
                  <button
                    onClick={handleRepay}
                    disabled={repayLoading}
                    className="w-full editorial-gradient text-white py-2.5 rounded-xl text-sm font-bold disabled:opacity-50"
                  >
                    {repayLoading ? "กำลังโหลด QR..." : "แสดง QR Code ชำระเงิน"}
                  </button>
                  {repayError && (
                    <p className="mt-2 text-xs text-error text-center">{repayError}</p>
                  )}
                </>
              )}
            </div>
          )}

          {/* Cancel section */}
          {(order.status === "PENDING" || order.status === "AWAITING_PAYMENT" || order.status === "CONFIRMED") && (
            <div className="bg-surface-container-lowest rounded-2xl p-7 border border-outline-variant/10">
              {!showCancel ? (
                <button
                  onClick={() => setShowCancel(true)}
                  className="w-full rounded-xl border border-error/30 py-2.5 text-sm font-semibold text-error transition-colors hover:bg-error/5"
                >
                  ยกเลิกคำสั่งซื้อ
                </button>
              ) : (
                <div className="space-y-3">
                  <p className="text-sm font-medium text-on-surface">ระบุเหตุผลในการยกเลิก</p>
                  <input
                    type="text"
                    value={cancelReason}
                    onChange={(e) => setCancelReason(e.target.value)}
                    placeholder="เช่น เปลี่ยนใจ / สั่งผิด"
                    className="w-full rounded-xl bg-surface-low/40 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-error/20"
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
                      className="flex-1 rounded-xl bg-error py-2 text-sm font-semibold text-white disabled:opacity-50"
                    >
                      {cancelling ? "กำลังยกเลิก..." : "ยืนยันยกเลิก"}
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </>
  );
}
