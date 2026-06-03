"use client";

import { useEffect, useState, useCallback } from "react";
import {
  Search,
  Eye,
  XCircle,
  ChevronLeft,
  ChevronRight,
  ShoppingCart,
} from "lucide-react";
import { cn, formatBaht } from "@/lib/utils";
import { adminOrderHistoryService, adminOrderService } from "@/lib/services";
import type { OrderHistory } from "@/lib/types";
import { APP_CONFIG } from "@/lib/constants";

const STATUS_MAP: Record<string, { label: string; cls: string }> = {
  PENDING: { label: "รอดำเนินการ", cls: "bg-amber-50 text-amber-700" },
  CONFIRMED: { label: "ยืนยันแล้ว", cls: "bg-blue-50 text-blue-700" },
  SHIPPED: { label: "จัดส่งแล้ว", cls: "bg-sky-50 text-sky-700" },
  COMPLETED: { label: "สำเร็จ", cls: "bg-emerald-50 text-emerald-700" },
  CANCELLED: { label: "ยกเลิก", cls: "bg-red-50 text-red-700" },
};

export default function AdminOrdersPage() {
  const [orders, setOrders] = useState<OrderHistory[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const limit = APP_CONFIG.PAGINATION.ADMIN_TABLE;

  const fetchOrders = useCallback(async () => {
    setLoading(true);
    try {
      const res = await adminOrderHistoryService.list(page, limit, statusFilter || undefined);
      setOrders(res.items ?? []);
      setTotal(res.total ?? 0);
    } catch {
      setOrders([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, statusFilter]);

  useEffect(() => {
    fetchOrders();
  }, [fetchOrders]);

  useEffect(() => { setPage(1); }, [statusFilter]);

  const handleCancel = async (id: string) => {
    if (!confirm("ต้องการยกเลิกคำสั่งซื้อนี้หรือไม่?")) return;
    await adminOrderService.cancel(id, "ยกเลิกโดยผู้ดูแลระบบ");
    fetchOrders();
  };

  /* ─── client filtering (text search only) ─── */
  const q = search.toLowerCase();
  const filtered = search
    ? orders.filter(
        (o) =>
          o.order_id?.toLowerCase().includes(q) ||
          o.shipping_address?.full_name?.toLowerCase().includes(q)
      )
    : orders;

  const totalPages = Math.ceil(total / limit) || 1;

  const getPaginationGroup = () => {
    const MAX_BUTTON = 5;
    let start = Math.max(1, page - 2);
    let end = Math.min(totalPages, start + MAX_BUTTON - 1);
    if (end - start + 1 < MAX_BUTTON) {
      start = Math.max(1, end - MAX_BUTTON + 1);
    }
    return Array.from({ length: end - start + 1 }, (_, i) => start + i);
  };

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight text-on-surface">
          คำสั่งซื้อทั้งหมด
        </h1>
        <p className="text-sm text-secondary">
          จัดการและติดตามคำสั่งซื้อ · ทั้งหมด {total} รายการ
        </p>
      </div>

      {/* Filters */}
      <div className="mb-6 flex items-center gap-3 rounded-2xl bg-white p-4 shadow-ambient">
        <div className="relative flex-1 max-w-sm">
          <Search
            size={16}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-outline"
          />
          <input
            type="text"
            placeholder="ค้นหา Order ID หรือชื่อลูกค้า..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg bg-surface-highest py-2 pl-9 pr-4 text-sm outline-none transition-all focus:ring-2 focus:ring-primary/20"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="rounded-lg bg-surface-highest px-3 py-2 text-sm text-secondary outline-none"
        >
          <option value="">ทุกสถานะ</option>
          {Object.entries(STATUS_MAP).map(([key, { label }]) => (
            <option key={key} value={key}>
              {label}
            </option>
          ))}
        </select>
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-2xl bg-white shadow-ambient">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-surface-highest bg-surface-low/30 text-xs uppercase tracking-wider text-secondary">
                <th className="px-6 py-4 font-medium">Order ID</th>
                <th className="px-4 py-4 font-medium">ลูกค้า</th>
                <th className="px-4 py-4 font-medium">วันที่</th>
                <th className="px-4 py-4 font-medium">จำนวนเงิน</th>
                <th className="px-4 py-4 font-medium">รายการ</th>
                <th className="px-4 py-4 font-medium">สถานะ</th>
                <th className="px-4 py-4 font-medium text-right">จัดการ</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-secondary">
                    กำลังโหลด...
                  </td>
                </tr>
              ) : filtered.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-16 text-center text-secondary">
                    <ShoppingCart
                      size={40}
                      className="mx-auto mb-2 text-outline"
                    />
                    ไม่พบคำสั่งซื้อ
                  </td>
                </tr>
              ) : (
                filtered.map((o) => {
                  const status = STATUS_MAP[o.status] ?? STATUS_MAP.PENDING;
                  return (
                    <tr
                      key={o.order_id}
                      className="border-b border-surface-highest/60 transition-colors hover:bg-surface-low/40"
                    >
                      {/* ID */}
                      <td className="px-6 py-4 font-mono text-xs font-medium text-on-surface">
                        #{o.order_id?.slice(0, 8)}
                      </td>
                      {/* Customer */}
                      <td className="px-4 py-4">
                        <div className="flex items-center gap-2">
                          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-container/40 text-xs font-bold text-primary">
                            {(
                              o.shipping_address?.full_name?.[0] ?? "U"
                            ).toUpperCase()}
                          </div>
                          <span className="text-sm text-on-surface">
                            {o.shipping_address?.full_name ?? "—"}
                          </span>
                        </div>
                      </td>
                      {/* Date */}
                      <td className="px-4 py-4 text-secondary">
                        {new Date(o.created_at).toLocaleDateString("th-TH", {
                          day: "numeric",
                          month: "short",
                          year: "2-digit",
                        })}
                      </td>
                      {/* Amount */}
                      <td className="px-4 py-4 font-medium text-on-surface">
                        {formatBaht(o.total_amount)}
                      </td>
                      {/* Items count */}
                      <td className="px-4 py-4 text-secondary">
                        {o.items?.length ?? 0} รายการ
                      </td>
                      {/* Status */}
                      <td className="px-4 py-4">
                        <span
                          className={cn(
                            "rounded-full px-3 py-1 text-xs font-medium",
                            status.cls
                          )}
                        >
                          {status.label}
                        </span>
                      </td>
                      {/* Actions */}
                      <td className="px-4 py-4 text-right">
                        <div className="flex items-center justify-end gap-1">
                          <a
                            href={`/orders/${o.order_id}`}
                            className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest hover:text-primary"
                            title="ดูรายละเอียด"
                          >
                            <Eye size={15} />
                          </a>
                          {o.status !== "CANCELLED" &&
                            o.status !== "COMPLETED" && (
                              <button
                                onClick={() => handleCancel(o.order_id)}
                                className="rounded-lg p-2 text-secondary transition-colors hover:bg-red-50 hover:text-error"
                                title="ยกเลิกคำสั่งซื้อ"
                              >
                                <XCircle size={15} />
                              </button>
                            )}
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between border-t border-surface-highest px-6 py-4">
          <p className="text-xs text-secondary">
            แสดง {(page - 1) * limit + 1}–{Math.min(page * limit, total)} จาก{" "}
            {total} รายการ
          </p>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
              className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest disabled:opacity-40"
            >
              <ChevronLeft size={16} />
            </button>
            {getPaginationGroup().map((n) => (
              <button
                key={n}
                onClick={() => setPage(n)}
                className={cn(
                  "h-8 w-8 rounded-lg text-xs font-medium transition-colors",
                  n === page
                    ? "bg-primary text-white"
                    : "text-secondary hover:bg-surface-highest"
                )}
              >
                {n}
              </button>
            ))}
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
              className="rounded-lg p-2 text-secondary transition-colors hover:bg-surface-highest disabled:opacity-40"
            >
              <ChevronRight size={16} />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
