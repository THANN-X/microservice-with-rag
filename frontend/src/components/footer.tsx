/**
 * Footer — footer ด้านล่างของทุกหน้า
 *
 * What: แสดง brand, links, ช่องทางชำระเงิน และ copyright
 * Why:  static component — ไม่มี state ใดเลย จึงไม่ใช้ "use client" (server component)
 * How:  new Date().getFullYear() — ปี copyright เปลี่ยนอัตโนมัติโดยไม่ต้อง hardcode
 */

import Link from "next/link";

export default function Footer() {
  return (
    <footer className="w-full mt-16 bg-surface-container-low">
      <div className="max-w-7xl mx-auto px-8 py-14">
        {/* Top Row */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-10 mb-12">
          {/* Brand */}
          <div className="md:col-span-1">
            <Link href="/" className="text-xl font-black text-primary tracking-tight">อนันตา</Link>
            <p className="text-sm text-on-surface-variant mt-3 leading-relaxed thai-line-height">
              คัดสรรสินค้าคุณภาพเพื่อคุณ<br />พร้อมบริการระดับพรีเมียม
            </p>
          </div>

          {/* Links */}
          <div>
            <h4 className="text-sm font-bold text-on-surface mb-4">เมนู</h4>
            <div className="flex flex-col gap-2.5">
              <Link href="/products" className="text-sm text-on-surface-variant hover:text-primary transition-colors">สินค้าทั้งหมด</Link>
              <Link href="/orders" className="text-sm text-on-surface-variant hover:text-primary transition-colors">คำสั่งซื้อ</Link>
              <Link href="/coupons" className="text-sm text-on-surface-variant hover:text-primary transition-colors">คูปอง</Link>
            </div>
          </div>

          <div>
            <h4 className="text-sm font-bold text-on-surface mb-4">ช่วยเหลือ</h4>
            <div className="flex flex-col gap-2.5">
              <Link href="#" className="text-sm text-on-surface-variant hover:text-primary transition-colors">คำถามที่พบบ่อย</Link>
              <Link href="#" className="text-sm text-on-surface-variant hover:text-primary transition-colors">นโยบายคืนสินค้า</Link>
              <Link href="/chat" className="text-sm text-on-surface-variant hover:text-primary transition-colors">ติดต่อเรา</Link>
            </div>
          </div>

          {/* Payment */}
          <div>
            <h4 className="text-sm font-bold text-on-surface mb-4">ช่องทางชำระเงิน</h4>
            <div className="flex gap-3">
              <div className="px-3 py-1.5 bg-surface-container-lowest rounded-md border border-outline-variant/20 text-xs font-bold text-on-surface-variant">PromptPay</div>
              <div className="px-3 py-1.5 bg-surface-container-lowest rounded-md border border-outline-variant/20 text-xs font-bold text-on-surface-variant">Visa</div>
              <div className="px-3 py-1.5 bg-surface-container-lowest rounded-md border border-outline-variant/20 text-xs font-bold text-on-surface-variant">MC</div>
            </div>
          </div>
        </div>

        {/* Bottom divider */}
        <div className="border-t border-surface-container-high pt-8 flex flex-col md:flex-row justify-between items-center gap-4">
          <p className="text-xs text-on-surface-variant">
            © {new Date().getFullYear()} อนันตา เอ็ดดิทอเรียล. สงวนลิขสิทธิ์.
          </p>
          <div className="flex gap-6">
            <Link href="#" className="text-xs text-on-surface-variant hover:text-primary transition-colors">ข้อกำหนดการใช้งาน</Link>
            <Link href="#" className="text-xs text-on-surface-variant hover:text-primary transition-colors">ความเป็นส่วนตัว</Link>
          </div>
        </div>
      </div>
    </footer>
  );
}
