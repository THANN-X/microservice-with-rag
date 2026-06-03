"use client";

import { Fragment } from "react";
import Image from "next/image";
import { useCart } from "@/context/cart-context";
import { formatBaht } from "@/lib/utils";
import Link from "next/link";
import { useRouter } from "next/navigation";

export default function CartPage() {
  const { cart, loading, updateItem, removeItem, updatingItems, isGuest } = useCart();
  const router = useRouter();

  if (loading) {
    return (
      <div className="flex justify-center py-20">
        <span className="material-symbols-outlined text-5xl text-outline animate-spin">progress_activity</span>
      </div>
    );
  }

  const items = cart?.items || [];
  const subtotal = items.reduce((sum, item) => sum + (item.price || 0) * item.quantity, 0);
  // #3 ตรวจสอบ item ที่ราคาเป็น 0 — อาจเกิดจาก metadata ขาดหาย
  const hasZeroPrice = items.some((item) => !item.price || item.price <= 0);
  const canCheckout = !isGuest && !hasZeroPrice && items.length > 0;

  const handleCheckout = () => {
    if (isGuest) {
      router.push("/login?callbackUrl=/checkout");
      return;
    }
    router.push("/checkout");
  };

  // What: กำหนด summary rows เป็น data — เพิ่ม discount/tax ได้โดย push เข้า array เพียงอย่างเดียว
  const summaryLines: { label: string; value: string; highlight?: boolean }[] = [
    { label: "ยอดรวมสินค้า", value: formatBaht(subtotal) },
    { label: "ค่าจัดส่ง",    value: "ฟรี", highlight: true },
  ];

  // What: กำหนด quantity control buttons เป็น data — เพิ่ม step/max ได้ง่าย
  const qtyActions = [
    { icon: "remove", calc: (q: number) => Math.max(1, q - 1) },
    { icon: "add",    calc: (q: number) => q + 1 },
  ] as const;

  return (
    <>
      <header className="mb-8">
        <h1 className="text-3xl font-black text-on-surface tracking-tight mb-1">ตะกร้าสินค้า</h1>
        <p className="text-on-surface-variant text-sm">{items.length} รายการในตะกร้า</p>
      </header>

      {/* Guest banner — แสดงเฉพาะ guest ที่มีสินค้าในตะกร้า */}
      {isGuest && items.length > 0 && (
        <div className="mb-6 flex items-center gap-3 bg-primary/5 border border-primary/15 rounded-xl px-5 py-3.5">
          <span className="material-symbols-outlined text-primary text-[22px]">info</span>
          <p className="text-sm text-on-surface-variant flex-1">
            ตะกร้าของคุณจะถูกบันทึกไว้เมื่อ{" "}
            <Link href="/login" className="text-primary font-bold hover:underline">เข้าสู่ระบบ</Link>
            {" "}— สินค้าจะถูกโอนอัตโนมัติ
          </p>
        </div>
      )}

      {items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 gap-5">
          <span className="material-symbols-outlined text-7xl text-outline">shopping_cart</span>
          <h2 className="text-xl font-bold text-on-surface">ตะกร้าว่างเปล่า</h2>
          <Link href="/products" className="editorial-gradient text-white px-7 py-3 rounded-xl font-bold text-sm">
            เริ่มช้อปปิ้ง
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-8">
          {/* Cart Items */}
          <div className="xl:col-span-2 space-y-4">
            {items.map((item) => {
              const isUpdating = updatingItems.has(item.variant_id);
              return (
                <div key={item.variant_id} className="bg-surface-container-lowest rounded-xl p-5 flex flex-col sm:flex-row gap-5 group transition-all duration-200 hover:shadow-md">
                  <div className="w-full sm:w-28 h-28 rounded-lg overflow-hidden bg-surface-container flex-shrink-0 relative">
                    {item.image_url ? (
                      <Image
                        alt={item.product_name || "สินค้า"}
                        src={item.image_url}
                        fill
                        className="object-cover"
                        sizes="112px"
                      />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center">
                        <span className="material-symbols-outlined text-3xl text-outline">image</span>
                      </div>
                    )}
                  </div>
                  <div className="flex-grow flex flex-col justify-between min-w-0">
                    <div className="flex justify-between items-start gap-2">
                      <div className="min-w-0">
                        <h3 className="font-bold text-on-surface truncate">{item.product_name || "สินค้า"}</h3>
                        {item.variant_name && <p className="text-xs text-on-surface-variant mt-0.5">{item.variant_name}</p>}
                      </div>
                      <button
                        onClick={() => removeItem(item.variant_id)}
                        disabled={isUpdating}
                        className="w-8 h-8 flex items-center justify-center rounded-lg text-on-surface-variant/50 hover:bg-error/5 hover:text-error transition-colors flex-shrink-0 disabled:opacity-40"
                      >
                        <span className={`material-symbols-outlined text-[20px] ${isUpdating ? "animate-spin" : ""}`}>
                          {isUpdating ? "progress_activity" : "close"}
                        </span>
                      </button>
                    </div>
                    <div className="flex justify-between items-end mt-4">
                      <div className="flex items-center bg-surface-container-highest rounded-lg">
                        {qtyActions.map((action, i) => (
                          <Fragment key={action.icon}>
                            {i === 1 && <span className="w-8 text-center text-sm font-bold">{item.quantity}</span>}
                            <button
                              disabled={isUpdating}
                              onClick={() => updateItem({ variantId: item.variant_id, quantity: action.calc(item.quantity) })}
                              className="w-9 h-9 flex items-center justify-center text-on-surface-variant hover:text-primary transition-colors disabled:opacity-40"
                            >
                              <span className="material-symbols-outlined text-[18px]">{action.icon}</span>
                            </button>
                          </Fragment>
                        ))}
                      </div>
                      <span className="text-lg font-black text-primary">
                        {formatBaht((item.price || 0) * item.quantity)}
                      </span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Order Summary */}
          <div className="xl:col-span-1">
            <div className="bg-surface-container-lowest rounded-2xl p-7 shadow-lg shadow-primary/5 sticky top-24">
              <h2 className="text-xl font-bold text-on-surface mb-6">สรุปคำสั่งซื้อ</h2>
              <div className="space-y-4 mb-6">
                {summaryLines.map((row) => (
                  <div key={row.label} className="flex justify-between items-center text-sm text-on-surface-variant">
                    <span>{row.label}</span>
                    <span className={row.highlight ? "text-primary font-bold" : "font-medium"}>{row.value}</span>
                  </div>
                ))}
                <div className="pt-4">
                  <label className="block text-xs font-bold text-on-surface-variant mb-2">
                    รหัสโปรโมชั่น
                  </label>
                  <div className="flex gap-2">
                    <input
                      className="flex-grow bg-surface-container-highest border-none rounded-lg px-3 py-2.5 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none"
                      placeholder="กรอกรหัสส่วนลด"
                      type="text"
                    />
                    <button className="bg-primary text-white px-4 py-2 rounded-lg text-sm font-bold hover:bg-primary-dim transition-colors">
                      ใช้งาน
                    </button>
                  </div>
                </div>
                <div className="pt-4 border-t border-surface-container flex justify-between items-end">
                  <span className="text-sm text-on-surface-variant">ยอดชำระสุทธิ</span>
                  <span className="text-2xl font-black text-primary">{formatBaht(subtotal)}</span>
                </div>
              </div>
              {/* #3 Zero-price warning */}
              {hasZeroPrice && (
                <div className="flex items-center gap-2 bg-error/5 border border-error/15 rounded-xl px-4 py-3 mb-4">
                  <span className="material-symbols-outlined text-error text-[20px]">warning</span>
                  <p className="text-xs text-error">สินค้าบางรายการมีราคาไม่ถูกต้อง กรุณาลบออกแล้วเพิ่มใหม่</p>
                </div>
              )}
              <button
                onClick={handleCheckout}
                disabled={items.length === 0 || hasZeroPrice}
                className="block w-full py-3.5 text-center editorial-gradient text-white rounded-xl font-bold text-sm shadow-lg shadow-primary/15 hover:shadow-xl hover:shadow-primary/25 transition-shadow disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none"
              >
                {isGuest ? "เข้าสู่ระบบเพื่อชำระเงิน" : "ดำเนินการชำระเงิน"}
              </button>
              <p className="text-[10px] text-center text-on-surface-variant mt-4 leading-relaxed">
                การคลิกปุ่ม &ldquo;ดำเนินการชำระเงิน&rdquo; แสดงว่าคุณยอมรับ
                <span className="underline ml-0.5">ข้อกำหนดและเงื่อนไข</span> ของอนันตา
              </p>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
