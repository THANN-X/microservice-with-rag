"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/context/auth-context";
import { orderHistoryService } from "@/lib/services";
import type { OrderHistory } from "@/lib/types";
import { formatBaht } from "@/lib/utils";
import Link from "next/link";
import { useRouter } from "next/navigation";

const statusMap: Record<string, { label: string; color: string }> = {
  PENDING: { label: "รอดำเนินการ", color: "bg-amber-100 text-amber-800" },
  CONFIRMED: { label: "ยืนยันแล้ว", color: "bg-blue-100 text-blue-800" },
  SHIPPED: { label: "กำลังจัดส่ง", color: "bg-secondary-container/30 text-on-secondary-container" },
  COMPLETED: { label: "จัดส่งสำเร็จ", color: "bg-tertiary-container/30 text-on-tertiary-container" },
  CANCELLED: { label: "ยกเลิก", color: "bg-error-container/30 text-on-error-container" },
};

export default function ProfilePage() {
  const { user, loading, logout, updateProfile, changePassword } = useAuth();
  const router = useRouter();

  // console.log("Check User Status:", { loading, user, role: user?.role });

  const [orders, setOrders] = useState<OrderHistory[]>([]);
  const [activeTab, setActiveTab] = useState("profile");

  // ─── Profile form state ───
  const [profileForm, setProfileForm] = useState({ first_name: "", last_name: "", phone: "", address: "" });
  const [profileSaving, setProfileSaving] = useState(false);
  const [profileMsg, setProfileMsg] = useState<{ type: "ok" | "err"; text: string } | null>(null);

  // ─── Password form state ───
  const [pwForm, setPwForm] = useState({ old_password: "", new_password: "", confirm: "" });
  const [pwSaving, setPwSaving] = useState(false);
  const [pwMsg, setPwMsg] = useState<{ type: "ok" | "err"; text: string } | null>(null);

  // Sync profile form when user data loads
  useEffect(() => {
    if (user) {
      setProfileForm({ first_name: user.first_name, last_name: user.last_name, phone: user.phone, address: user.address });
    }
  }, [user]);

  useEffect(() => {
    if (user) {
      orderHistoryService.list(1, 5).then((res) => setOrders(res.items || [])).catch(() => {});
    }
  }, [user]);

    // Why: หน้านี้ใช้ email ซึ่งมีเฉพาะ UserProfile — admin ไม่มี profile หน้านี้
  useEffect(() => {
    if (!loading && user && user.role !== "customer") {
      router.replace("/");
    }
  }, [loading, user, router]);

  

  const handleProfileSave = async () => {
    if (!profileForm.first_name.trim() || !profileForm.last_name.trim()) {
      setProfileMsg({ type: "err", text: "กรุณากรอกชื่อและนามสกุล" });
      return;
    }
    if (!profileForm.phone.trim()) {
      setProfileMsg({ type: "err", text: "กรุณากรอกเบอร์โทรศัพท์" });
      return;
    }
    if (!profileForm.address.trim()) {
      setProfileMsg({ type: "err", text: "กรุณากรอกที่อยู่จัดส่ง" });
      return;
    }

    setProfileSaving(true);
    setProfileMsg(null);
    try {
      await updateProfile(profileForm);
      setProfileMsg({ type: "ok", text: "บันทึกข้อมูลเรียบร้อย" });
    } catch (e) {
      let errorText = "เกิดข้อผิดพลาดในการบันทึกข้อมูล";
      
      if (e instanceof Error) {
        const rawMsg = e.message.toLowerCase();
        
        // ดักคีย์เวิร์ดจาก Backend แล้วแปล
        if (rawMsg.includes("phone")) {
          errorText = "เบอร์โทรศัพท์ไม่ถูกต้อง หรือกรอกไม่ครบครับ";
        } else if (rawMsg.includes("address")) {
          errorText = "กรุณากรอกที่อยู่ให้ชัดเจนด้วยนะครับ";
        } else if (rawMsg.includes("required")) {
          errorText = "กรุณากรอกข้อมูลให้ครบถ้วนครับ";
        } else if (rawMsg.includes("400")) {
          errorText = "ข้อมูลไม่ถูกต้อง กรุณาตรวจสอบอีกครั้ง (400)";
        } else {
          errorText = e.message; // ถ้าดักไม่ตรงเลย ก็แสดงของเดิมไป
        }
      }
      
      setProfileMsg({ type: "err", text: errorText });
    } finally {
      setProfileSaving(false);
    }
  };

  const handlePasswordChange = async () => {
    if (pwForm.new_password !== pwForm.confirm) {
      setPwMsg({ type: "err", text: "รหัสผ่านใหม่ไม่ตรงกัน" });
      return;
    }
    setPwSaving(true);
    setPwMsg(null);
    try {
      await changePassword({ old_password: pwForm.old_password, new_password: pwForm.new_password });
      setPwMsg({ type: "ok", text: "เปลี่ยนรหัสผ่านเรียบร้อย" });
      setPwForm({ old_password: "", new_password: "", confirm: "" });
    } catch (e) {
      let errorText = "เกิดข้อผิดพลาดในการเปลี่ยนรหัสผ่าน";
      
      if (e instanceof Error) {
        const rawMsg = e.message.toLowerCase();
        
        // ดักคีย์เวิร์ดจาก Backend แล้วแปล
        if (rawMsg.includes("old_password")) {
          errorText = "รหัสผ่านเก่าไม่ถูกต้อง";
        } else if (rawMsg.includes("new_password")) {
          errorText = "รหัสผ่านใหม่ไม่ถูกต้อง";
        } else if (rawMsg.includes("required")) {
          errorText = "กรุณากรอกข้อมูลให้ครบถ้วนครับ";
        } else if (rawMsg.includes("400")) {
          errorText = "ข้อมูลไม่ถูกต้อง กรุณาตรวจสอบอีกครั้ง (400)";
        } else {
          errorText = e.message; // ถ้าดักไม่ตรงเลย ก็แสดงของเดิมไป
        }
      }
      
      setPwMsg({ type: "err", text: errorText });
    } finally {
      setPwSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="space-y-5">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-5 mb-8">
          <div className="md:col-span-2 bg-surface-container-lowest rounded-2xl p-7 flex items-center gap-5 border border-outline-variant/10 animate-pulse">
            <div className="w-16 h-16 rounded-full bg-surface-container-high" />
            <div className="flex-1 space-y-2">
              <div className="h-5 bg-surface-container-high rounded-lg w-40" />
              <div className="h-3.5 bg-surface-container-high rounded-lg w-56" />
            </div>
          </div>
          <div className="editorial-gradient rounded-2xl p-7 opacity-30 animate-pulse" />
        </div>
        <div className="h-12 bg-surface-container-low rounded-xl animate-pulse" />
        <div className="bg-surface-container-lowest rounded-2xl p-7 border border-outline-variant/10 animate-pulse space-y-4">
          <div className="h-5 w-32 bg-surface-container-high rounded-lg" />
          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-12 bg-surface-container-high rounded-xl" />
            ))}
          </div>
        </div>
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
      {/* Profile Header Bento */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-5 mb-8">
        <div className="md:col-span-2 bg-surface-container-lowest rounded-2xl p-7 flex items-center gap-5 border border-outline-variant/10">
          <div className="w-16 h-16 rounded-full editorial-gradient flex items-center justify-center text-white text-2xl font-black shadow-lg">
            {user.first_name?.charAt(0)?.toUpperCase() || "?"}
          </div>
          <div className="flex-1 min-w-0">
            <h1 className="text-xl font-black text-on-surface truncate">{user.first_name} {user.last_name}</h1>
            <p className="text-on-surface-variant text-sm">{'email' in user ? user.email : user.username}</p>
            {user.phone && <p className="text-on-surface-variant text-xs mt-0.5">{user.phone}</p>}
          </div>
          <button
            onClick={logout}
            className="text-xs font-bold text-error border border-error/20 px-5 py-2 rounded-xl hover:bg-error/5 transition-colors shrink-0"
          >
            ออกจากระบบ
          </button>
        </div>
        <div className="editorial-gradient rounded-2xl p-7 text-white flex flex-col justify-center">
          <p className="text-[10px] uppercase tracking-widest opacity-70">กระเป๋าเงิน</p>
          <p className="text-2xl font-black mt-1.5">฿ 0.00</p>
          <p className="text-xs opacity-70 mt-1">ยอดเงินคงเหลือ</p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-surface-container-low p-1 rounded-xl mb-8 overflow-x-auto no-scrollbar">
        {[
          { key: "profile", label: "โปรไฟล์", icon: "person" },
          { key: "address", label: "ที่อยู่", icon: "location_on" },
          { key: "orders", label: "คำสั่งซื้อ", icon: "receipt_long" },
          { key: "security", label: "ความปลอดภัย", icon: "shield" },
        ].map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`flex items-center gap-2 px-5 py-2.5 rounded-lg text-sm whitespace-nowrap transition-all ${
              activeTab === tab.key
                ? "bg-surface-container-lowest text-primary font-bold shadow-sm"
                : "text-on-surface-variant hover:text-on-surface"
            }`}
          >
            <span className="material-symbols-outlined text-lg" style={activeTab === tab.key ? { fontVariationSettings: "'FILL' 1" } : undefined}>{tab.icon}</span>
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {activeTab === "profile" && (
        <div className="bg-surface-container-lowest rounded-2xl p-7 space-y-5 border border-outline-variant/10">
          <h2 className="text-lg font-bold text-on-surface">ข้อมูลส่วนตัว</h2>
          {profileMsg && (
            <p className={`text-sm font-medium px-4 py-2.5 rounded-xl ${profileMsg.type === "ok" ? "bg-tertiary-container/30 text-on-tertiary-container" : "bg-error-container/20 text-error"}`}>
              {profileMsg.text}
            </p>
          )}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            <div>
              <label className="block text-xs font-semibold mb-1.5 text-on-surface-variant">ชื่อ</label>
              <input
                type="text"
                value={profileForm.first_name}
                onChange={(e) => setProfileForm({ ...profileForm, first_name: e.target.value })}
                className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-3.5 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold mb-1.5 text-on-surface-variant">นามสกุล</label>
              <input
                type="text"
                value={profileForm.last_name}
                onChange={(e) => setProfileForm({ ...profileForm, last_name: e.target.value })}
                className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-3.5 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold mb-1.5 text-on-surface-variant">อีเมล</label>
              <input
                type="email"
                value={'email' in user ? user.email : user.username}
                className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-3.5 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none opacity-60"
                readOnly
              />
            </div>
            <div>
              <label className="block text-xs font-semibold mb-1.5 text-on-surface-variant">เบอร์โทรศัพท์</label>
              <input
                type="text"
                value={profileForm.phone}
                onChange={(e) => setProfileForm({ ...profileForm, phone: e.target.value })}
                className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-3.5 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none"
              />
            </div>
            <div className="md:col-span-2">
              <label className="block text-xs font-semibold mb-1.5 text-on-surface-variant">ที่อยู่</label>
              <input
                type="text"
                value={profileForm.address}
                onChange={(e) => setProfileForm({ ...profileForm, address: e.target.value })}
                placeholder="ที่อยู่สำหรับจัดส่ง"
                className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-3.5 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none"
              />
            </div>
          </div>
          <button
            onClick={handleProfileSave}
            disabled={profileSaving}
            className="editorial-gradient text-white text-sm font-bold px-6 py-3 rounded-xl mt-2 disabled:opacity-50"
          >
            {profileSaving ? "กำลังบันทึก..." : "บันทึกข้อมูล"}
          </button>
        </div>
      )}

      {activeTab === "orders" && (
        <section className="space-y-5">
          <div className="flex justify-between items-end">
            <h2 className="text-lg font-bold text-on-surface">ประวัติการสั่งซื้อล่าสุด</h2>
            <Link href="/orders" className="text-primary font-bold text-xs hover:underline">ดูทั้งหมด</Link>
          </div>
          <div className="space-y-3">
            {orders.length === 0 && (
              <p className="text-center text-on-surface-variant py-10">ยังไม่มีคำสั่งซื้อ</p>
            )}
            {orders.map((order) => {
              const status = statusMap[order.status] || statusMap.PENDING;
              return (
                <Link
                  key={order.order_id}
                  href={`/orders/${order.order_id}`}
                  className="bg-surface-container-lowest rounded-2xl p-5 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:shadow-lg hover:shadow-primary/5 transition-all border border-outline-variant/10 block"
                >
                  <div className="flex gap-4">
                    <div className="w-14 h-14 bg-surface-container-low rounded-xl shrink-0 flex items-center justify-center">
                      <span className="material-symbols-outlined text-xl text-outline">receipt_long</span>
                    </div>
                    <div className="space-y-0.5">
                      <span className={`text-[11px] font-bold px-2.5 py-0.5 rounded-full ${status.color}`}>
                        {status.label}
                      </span>
                      <h4 className="font-bold text-sm text-on-surface">คำสั่งซื้อ #{order.order_id.slice(0, 8)}</h4>
                      <p className="text-xs text-on-surface-variant">
                        วันที่: {new Date(order.created_at).toLocaleDateString("th-TH")}
                      </p>
                    </div>
                  </div>
                  <div className="flex flex-row md:flex-col items-center md:items-end justify-between md:justify-center gap-1.5">
                    <p className="text-lg font-black text-primary leading-none">{formatBaht(order.total_amount)}</p>
                    <span className="text-xs font-bold text-primary flex items-center gap-1">
                      ดูรายละเอียด
                      <span className="material-symbols-outlined text-sm">chevron_right</span>
                    </span>
                  </div>
                </Link>
              );
            })}
          </div>
        </section>
      )}

      {activeTab === "address" && (
        <div className="bg-surface-container-lowest rounded-2xl p-7 border border-outline-variant/10">
          <div className="flex justify-between items-center mb-5">
            <h2 className="text-lg font-bold text-on-surface">ที่อยู่จัดส่ง</h2>
          </div>
          {user.address ? (
            <div className="bg-surface-container-low rounded-xl p-5">
              <p className="text-sm text-on-surface">{user.address}</p>
            </div>
          ) : (
            <div className="flex flex-col items-center py-10 gap-3">
              <span className="material-symbols-outlined text-5xl text-outline">add_location</span>
              <p className="text-sm text-on-surface-variant">ยังไม่ได้ตั้งค่าที่อยู่จัดส่ง</p>
            </div>
          )}
        </div>
      )}

      {activeTab === "security" && (
        <div className="bg-surface-container-lowest rounded-2xl p-7 space-y-5 border border-outline-variant/10">
          <h2 className="text-lg font-bold text-on-surface">เปลี่ยนรหัสผ่าน</h2>
          {pwMsg && (
            <p className={`text-sm font-medium px-4 py-2.5 rounded-xl ${pwMsg.type === "ok" ? "bg-tertiary-container/30 text-on-tertiary-container" : "bg-error-container/20 text-error"}`}>
              {pwMsg.text}
            </p>
          )}
          <div className="space-y-4 max-w-md">
            <div>
              <label className="block text-xs font-semibold mb-1.5 text-on-surface-variant">รหัสผ่านปัจจุบัน</label>
              <input
                type="password"
                value={pwForm.old_password}
                onChange={(e) => setPwForm({ ...pwForm, old_password: e.target.value })}
                placeholder="กรอกรหัสผ่านปัจจุบัน"
                className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-3.5 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold mb-1.5 text-on-surface-variant">รหัสผ่านใหม่</label>
              <input
                type="password"
                value={pwForm.new_password}
                onChange={(e) => setPwForm({ ...pwForm, new_password: e.target.value })}
                placeholder="รหัสผ่านใหม่"
                className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-3.5 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold mb-1.5 text-on-surface-variant">ยืนยันรหัสผ่านใหม่</label>
              <input
                type="password"
                value={pwForm.confirm}
                onChange={(e) => setPwForm({ ...pwForm, confirm: e.target.value })}
                placeholder="ยืนยันรหัสผ่านใหม่"
                className="w-full bg-surface-container-highest border-none rounded-xl px-4 py-3.5 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none"
              />
            </div>
            <button
              onClick={handlePasswordChange}
              disabled={pwSaving || !pwForm.old_password || !pwForm.new_password || !pwForm.confirm}
              className="editorial-gradient text-white text-sm font-bold px-6 py-3 rounded-xl mt-1 disabled:opacity-50"
            >
              {pwSaving ? "กำลังเปลี่ยน..." : "เปลี่ยนรหัสผ่าน"}
            </button>
          </div>
        </div>
      )}
    </>
  );
}
