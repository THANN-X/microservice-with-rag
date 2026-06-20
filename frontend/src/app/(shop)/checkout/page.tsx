"use client";

import { useState, useEffect } from "react";
import Image from "next/image";
import { useCart } from "@/context/cart-context";
import { useAuth } from "@/context/auth-context";
import { orderService } from "@/lib/services";
import { formatBaht } from "@/lib/utils";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { loadStripe } from "@stripe/stripe-js";
import {
  Elements,
  CardElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";

// loadStripe ไว้นอก component เพื่อไม่ให้ re-create ทุก render
const stripePromise = loadStripe(
  process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY || ""
);

type PaymentMethod = "promptpay" | "credit";

// ─── Inner checkout form (ต้องอยู่ใน <Elements> provider) ───────────────────
function CheckoutForm() {
  const { cart, clearCart } = useCart();
  const { user, loading } = useAuth();
  const router = useRouter();
  const stripe = useStripe();
  const elements = useElements();

  const [payment, setPayment] = useState<PaymentMethod>("promptpay");
  const [address, setAddress] = useState({
    full_name: "",
    phone: "",
    address_line: "",
    sub_district: "",
    district: "",
    province: "กรุงเทพมหานคร",
    postal_code: "",
  });
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Sync address จาก user เมื่อ auth โหลดเสร็จ
  useEffect(() => {
    if (user) {
      setAddress((prev) => ({
        ...prev,
        full_name: prev.full_name || `${user.first_name} ${user.last_name}`.trim(),
        phone: prev.phone || user.phone || "",
      }));
    }
  }, [user]);

  // PromptPay QR state
  const [qrImageUrl, setQrImageUrl] = useState<string | null>(null);
  const [awaitingOrderId, setAwaitingOrderId] = useState<string | null>(null);
  const [qrExpired, setQrExpired] = useState(false);
  const [pollFailed, setPollFailed] = useState(false);

  // Idempotency: เก็บ orderId ที่รอชำระเงินอยู่ → ถ้า payment fail แล้ว retry จะใช้ order เดิม ไม่สร้างใหม่
  const [pendingOrderId, setPendingOrderId] = useState<string | null>(null);

  // Card validation: true เมื่อ CardElement กรอกครบและถูกต้อง
  const [isCardComplete, setIsCardComplete] = useState(false);

  // Polling: ตรวจสอบสถานะ order ทุก 4 วินาที หลังแสดง QR PromptPay
  useEffect(() => {
    if (!awaitingOrderId) return;

    let isMounted = true; // ป้องกัน setState หลัง unmount
    let consecutiveErrors = 0;
    const MAX_CONSECUTIVE_ERRORS = 3;
    const MAX_ATTEMPTS = 225; // 15 นาที (225 × 4 วินาที)
    let attempts = 0;

    const intervalId = setInterval(async () => {
      attempts++;

      if (attempts > MAX_ATTEMPTS) {
        clearInterval(intervalId);
        if (isMounted) setQrExpired(true);
        return;
      }

      try {
        const order = await orderService.get(awaitingOrderId);
        if (!isMounted) return; // component unmount ระหว่าง await
        consecutiveErrors = 0;

        if (order.status === "PAID") {
          clearInterval(intervalId);
          router.push(`/orders/${awaitingOrderId}`);
        } else if (order.status === "CANCELLED") {
          clearInterval(intervalId);
          setError("คำสั่งซื้อถูกยกเลิก — หมดเวลาชำระเงิน");
          setQrImageUrl(null);
          setAwaitingOrderId(null);
        }
      } catch {
        if (!isMounted) return;
        consecutiveErrors++;
        if (consecutiveErrors >= MAX_CONSECUTIVE_ERRORS) {
          clearInterval(intervalId);
          setPollFailed(true);
        }
      }
    }, 4000);

    return () => {
      isMounted = false;
      clearInterval(intervalId);
    };
  }, [awaitingOrderId, router]);

  const items = cart?.items || [];
  const subtotal = items.reduce((sum, i) => sum + (i.price || 0) * i.quantity, 0);
  const shipping = 50;
  const total = subtotal + shipping;

  // snapshot ยอดรวมก่อน clear cart (ใช้แสดงบน QR screen หลัง clearCart() แล้ว)
  const [snapshotTotal, setSnapshotTotal] = useState<number>(0);

  const canSubmit =
    !!stripe &&
    !!elements &&
    !!address.full_name &&
    !!address.phone &&
    !!address.address_line &&
    !!address.postal_code &&
    !submitting &&
    items.length > 0 &&
    subtotal > 0 &&
    (payment === "promptpay" || (payment === "credit" && isCardComplete));

  const handleAddressChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setAddress((prev) => ({ ...prev, [name]: value }));
  };

  // รอ order ออกจาก PENDING (Kafka Saga: product_service reserve stock)
  // poll ทุก 500ms สูงสุด 10 วินาที (20 ครั้ง) — Saga ปกติใช้เวลา <1 วินาที
  // Returns: status ล่าสุดที่ไม่ใช่ PENDING
  const waitForConfirmed = async (orderId: string): Promise<import("@/lib/types").OrderStatus> => {
    const MAX_ATTEMPTS = 20;
    const INTERVAL_MS = 500;
    for (let i = 0; i < MAX_ATTEMPTS; i++) {
      const order = await orderService.get(orderId);
      if (order.status !== "PENDING") return order.status;
      await new Promise((r) => setTimeout(r, INTERVAL_MS));
    }
    throw new Error("ระบบใช้เวลานานเกินปกติ กรุณาลองใหม่");
  };

  const handleSubmit = async () => {
    if (!stripe || !elements) return;
    setSubmitting(true);
    setError(null);

    try {
      // Idempotency: ถ้ามี order ค้างอยู่ (payment fail ก่อนหน้า) → ใช้ order เดิม ไม่สร้างใหม่
      let orderId = pendingOrderId;
      if (!orderId) {
        const orderRes = await orderService.create({
          items: items.map((i) => ({
            variant_id: i.variant_id,
            quantity: i.quantity,
            // unit_price ไม่ส่ง — order_service ดึงราคาจาก catalog_service เอง (ป้องกัน price tampering)
          })),
          shipping_address: address,
          note: "",
        });
        orderId = orderRes.order.id;
        setPendingOrderId(orderId);
      }

      // รอ Kafka Saga: PENDING → CONFIRMED (product_service reserve stock)
      // WHY ต้องรอ? — ProcessPayment ต้องการ status == CONFIRMED
      // Saga ใช้เวลา ~200-800ms (Kafka round trip + product_service)
      const orderStatus = await waitForConfirmed(orderId);

      if (orderStatus === "CANCELLED") {
        setPendingOrderId(null);
        throw new Error("สินค้าในคำสั่งซื้อหมดสต็อก กรุณาลองใหม่");
      }
      // PAID/AWAITING_PAYMENT/COMPLETED → order ถูกชำระไปแล้ว (retry กับ order เก่า)
      if (orderStatus !== "CONFIRMED") {
        setPendingOrderId(null);
        router.push(`/orders/${orderId}`);
        return;
      }

      if (payment === "credit") {
        await handleCreditCard(orderId, stripe, elements);
        // credit card สำเร็จ → router.push ถูกเรียกแล้ว — ไม่ setSubmitting(false)
        // เพื่อให้ปุ่มยัง disabled จนกว่า Next.js เปลี่ยนหน้าเสร็จ
      } else {
        await handlePromptPay(orderId);
        setSubmitting(false); // PromptPay แสดง QR อยู่หน้าเดิม — คืนปุ่ม
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "เกิดข้อผิดพลาด กรุณาลองใหม่");
      setSubmitting(false); // error — คืนปุ่มให้กดใหม่ได้
      // ไม่ clear pendingOrderId → retry ครั้งถัดไปจะใช้ order เดิม
    }
  };

  // ─── Credit Card flow (3DS-compatible) ────────────────────────────────────
  // 1. Backend สร้าง PaymentIntent (ไม่ confirm) → คืน client_secret
  // 2. Frontend เรียก stripe.confirmCardPayment(client_secret, {card})
  //    → Stripe.js เด้ง OTP popup อัตโนมัติถ้าบัตรต้องการ 3DS
  // 3. ถ้า succeeded → redirect
  const handleCreditCard = async (
    orderId: string,
    stripeInstance: ReturnType<typeof useStripe> & object,
    elementsInstance: ReturnType<typeof useElements> & object
  ) => {
    const cardElement = (elementsInstance as ReturnType<typeof useElements>)!.getElement(CardElement);
    if (!cardElement) throw new Error("Card element not ready");

    // 1. Backend สร้าง PaymentIntent → ได้ client_secret
    const payRes = await orderService.processPayment(orderId, {
      token: "", // ไม่ส่ง pm_xxxx — frontend จะ confirm ด้วย card element โดยตรง
      payment_method: "CREDIT_CARD",
    });

    const clientSecret = payRes.payment.client_secret;
    if (!clientSecret) throw new Error("ไม่ได้รับ client_secret จาก server");

    // 2. confirmCardPayment — Stripe.js จัดการ 3DS / OTP popup เอง
    const { error: confirmError, paymentIntent } = await (stripeInstance as ReturnType<typeof useStripe>)!.confirmCardPayment(
      clientSecret,
      { payment_method: { card: cardElement } }
    );

    if (confirmError) throw new Error(confirmError.message);

    if (paymentIntent?.status === "succeeded" || paymentIntent?.status === "processing") {
      // "processing" = เครือข่ายบางแห่งยังประมวลผลอยู่ แต่ถือว่าลูกค้าชำระแล้ว
      setPendingOrderId(null);
      await clearCart();
      router.push(`/orders/${orderId}`);
    } else {
      throw new Error("การชำระเงินไม่สำเร็จ กรุณาลองใหม่");
    }
  };

  // ─── PromptPay flow ────────────────────────────────────────────────────────
  // Backend สร้าง PaymentMethod + confirm PI + คืน QR URL มาแล้ว
  // Frontend แค่แสดงรูป ไม่ต้องเรียก Stripe.js confirm
  const handlePromptPay = async (orderId: string) => {
    const payRes = await orderService.processPayment(orderId, {
      token: "",
      payment_method: "PROMPTPAY",
    });

    const qrUrl = payRes.payment.qr_image_url;
    if (!qrUrl) throw new Error("ไม่ได้รับ QR Code จาก server กรุณาลองใหม่");

    setSnapshotTotal(total); // snapshot ยอดก่อน clear cart
    setPendingOrderId(null);
    await clearCart();
    setQrImageUrl(qrUrl);
    setAwaitingOrderId(orderId);
  };

  // ─── PromptPay QR Screen ───────────────────────────────────────────────────
  if (qrImageUrl && awaitingOrderId) {
    // QR หมดอายุ (>15 นาที)
    if (qrExpired) {
      return (
        <div className="flex flex-col items-center gap-6 py-16 max-w-sm mx-auto text-center">
          <span className="material-symbols-outlined text-5xl text-error" style={{ fontVariationSettings: "'FILL' 1" }}>
            timer_off
          </span>
          <h2 className="text-2xl font-black text-on-surface">QR Code หมดอายุ</h2>
          <p className="text-sm text-on-surface-variant">เกินเวลา 15 นาที กรุณาทำรายการใหม่อีกครั้ง</p>
          <button
            onClick={() => {
              setQrImageUrl(null);
              setAwaitingOrderId(null);
              setQrExpired(false);
              setPollFailed(false);
            }}
            className="editorial-gradient text-white px-8 py-3 rounded-full font-bold text-sm"
          >
            ทำรายการใหม่
          </button>
          <Link href={`/orders/${awaitingOrderId}`} className="text-xs text-on-surface-variant hover:underline">
            ดูสถานะคำสั่งซื้อเดิม →
          </Link>
        </div>
      );
    }

    return (
      <div className="flex flex-col items-center gap-6 py-16 max-w-sm mx-auto text-center">
        <span className="material-symbols-outlined text-5xl text-primary" style={{ fontVariationSettings: "'FILL' 1" }}>
          qr_code_2
        </span>
        <h2 className="text-2xl font-black text-on-surface">สแกน QR เพื่อชำระเงิน</h2>
        <p className="text-sm text-on-surface-variant">ยอดชำระ <span className="font-bold text-primary">{formatBaht(snapshotTotal)}</span></p>
        <div className="bg-white p-4 rounded-2xl shadow-lg border border-outline-variant/20">
          <Image src={qrImageUrl} alt="PromptPay QR Code" width={240} height={240} />
        </div>
        {pollFailed ? (
          <div className="space-y-3">
            <p className="text-xs text-error">ไม่สามารถตรวจสอบสถานะอัตโนมัติได้ชั่วคราว</p>
            <Link
              href={`/orders/${awaitingOrderId}`}
              className="inline-block editorial-gradient text-white px-6 py-2.5 rounded-full font-bold text-sm"
            >
              ตรวจสอบสถานะการชำระเงิน →
            </Link>
          </div>
        ) : (
          <p className="text-xs text-on-surface-variant leading-relaxed">
            เปิดแอปธนาคารหรือ Mobile Banking → สแกน QR<br />
            หลังชำระเงินสำเร็จ คำสั่งซื้อจะอัปเดตอัตโนมัติ
          </p>
        )}
        {!pollFailed && (
          <Link
            href={`/orders/${awaitingOrderId}`}
            className="text-sm font-bold text-primary hover:underline"
          >
            ดูสถานะคำสั่งซื้อ →
          </Link>
        )}
      </div>
    );
  }

  if (loading) {
    return (
      <div className="animate-pulse space-y-8">
        <div className="h-8 bg-surface-container-low rounded-lg w-64" />
        <div className="h-80 bg-surface-container-low rounded-2xl" />
        <div className="h-48 bg-surface-container-low rounded-2xl" />
      </div>
    );
  }

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-6">
        <span className="material-symbols-outlined text-7xl text-outline">lock</span>
        <h1 className="text-3xl font-black text-on-surface">กรุณาเข้าสู่ระบบ</h1>
        <Link href={`/login?callbackUrl=${encodeURIComponent("/checkout")}`} className="editorial-gradient text-white px-8 py-4 rounded-full font-bold">
          เข้าสู่ระบบ
        </Link>
      </div>
    );
  }

  // Admin ไม่สามารถสั่งซื้อสินค้าได้ — narrow type ให้ส่วนที่เหลือของ component รู้ว่าเป็น customer
  if (user.role !== "customer") {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-6">
        <span className="material-symbols-outlined text-7xl text-outline">admin_panel_settings</span>
        <h1 className="text-3xl font-black text-on-surface">ไม่สามารถสั่งซื้อได้</h1>
        <p className="text-on-surface-variant text-sm">บัญชี Admin ไม่รองรับการสั่งซื้อสินค้า</p>
      </div>
    );
  }

  return (
    <>
      <header className="mb-10">
        <h1 className="text-3xl font-black text-on-surface tracking-tight mb-1">ขั้นตอนการชำระเงิน</h1>
        <p className="text-on-surface-variant text-sm">ตรวจสอบข้อมูลการสั่งซื้อและเลือกช่องทางการชำระเงิน</p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* Left Column */}
        <div className="lg:col-span-8 space-y-8">
          {/* Shipping Address */}
          <section className="bg-surface-container-lowest p-7 rounded-2xl">
            <div className="flex items-center gap-3 mb-6">
              <span className="material-symbols-outlined text-primary text-[22px]" style={{ fontVariationSettings: "'FILL' 1" }}>location_on</span>
              <h2 className="text-lg font-bold text-on-surface">ข้อมูลที่อยู่จัดส่ง</h2>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="md:col-span-2">
                <label className="block text-sm font-semibold mb-2 text-on-surface-variant">ชื่อ-นามสกุล ผู้รับ</label>
                <input type="text" name="full_name" value={address.full_name} onChange={handleAddressChange}
                  className="w-full bg-surface-container-highest border-none rounded-lg px-4 py-4 focus:ring-2 focus:ring-primary/20 transition-all outline-none" placeholder="ระบุชื่อจริงและนามสกุล" />
              </div>
              <div className="md:col-span-2">
                <label className="block text-sm font-semibold mb-2 text-on-surface-variant">ที่อยู่</label>
                <textarea name="address_line" value={address.address_line} onChange={handleAddressChange}
                  className="w-full bg-surface-container-highest border-none rounded-lg px-4 py-4 focus:ring-2 focus:ring-primary/20 transition-all outline-none" placeholder="ระบุรายละเอียดที่อยู่" rows={3} />
              </div>
              <div>
                <label className="block text-sm font-semibold mb-2 text-on-surface-variant">เบอร์โทรศัพท์</label>
                <input type="tel" name="phone" value={address.phone} onChange={handleAddressChange}
                  maxLength={10}
                  className="w-full bg-surface-container-highest border-none rounded-lg px-4 py-4 focus:ring-2 focus:ring-primary/20 transition-all outline-none" placeholder="08X-XXX-XXXX" />
              </div>
              <div>
                <label className="block text-sm font-semibold mb-2 text-on-surface-variant">แขวง/ตำบล</label>
                <input type="text" name="sub_district" value={address.sub_district} onChange={handleAddressChange}
                  className="w-full bg-surface-container-highest border-none rounded-lg px-4 py-4 focus:ring-2 focus:ring-primary/20 transition-all outline-none" placeholder="แขวง/ตำบล" />
              </div>
              <div>
                <label className="block text-sm font-semibold mb-2 text-on-surface-variant">เขต/อำเภอ</label>
                <input type="text" name="district" value={address.district} onChange={handleAddressChange}
                  className="w-full bg-surface-container-highest border-none rounded-lg px-4 py-4 focus:ring-2 focus:ring-primary/20 transition-all outline-none" placeholder="เขต/อำเภอ" />
              </div>
              <div>
                <label className="block text-sm font-semibold mb-2 text-on-surface-variant">จังหวัด</label>
                <select name="province" value={address.province} onChange={handleAddressChange}
                  className="w-full bg-surface-container-highest border-none rounded-lg px-4 py-4 focus:ring-2 focus:ring-primary/20 transition-all outline-none">
                  <option>กรุงเทพมหานคร</option>
                  <option>กระบี่</option>
                  <option>กาญจนบุรี</option>
                  <option>กาฬสินธุ์</option>
                  <option>กำแพงเพชร</option>
                  <option>ขอนแก่น</option>
                  <option>จันทบุรี</option>
                  <option>ฉะเชิงเทรา</option>
                  <option>ชลบุรี</option>
                  <option>ชัยนาท</option>
                  <option>ชัยภูมิ</option>
                  <option>ชุมพร</option>
                  <option>เชียงราย</option>
                  <option>เชียงใหม่</option>
                  <option>ตรัง</option>
                  <option>ตราด</option>
                  <option>ตาก</option>
                  <option>นครนายก</option>
                  <option>นครปฐม</option>
                  <option>นครพนม</option>
                  <option>นครราชสีมา</option>
                  <option>นครศรีธรรมราช</option>
                  <option>นครสวรรค์</option>
                  <option>นนทบุรี</option>
                  <option>นราธิวาส</option>
                  <option>น่าน</option>
                  <option>บึงกาฬ</option>
                  <option>บุรีรัมย์</option>
                  <option>ปทุมธานี</option>
                  <option>ประจวบคีรีขันธ์</option>
                  <option>ปราจีนบุรี</option>
                  <option>ปัตตานี</option>
                  <option>พระนครศรีอยุธยา</option>
                  <option>พะเยา</option>
                  <option>พังงา</option>
                  <option>พัทลุง</option>
                  <option>พิจิตร</option>
                  <option>พิษณุโลก</option>
                  <option>เพชรบุรี</option>
                  <option>เพชรบูรณ์</option>
                  <option>แพร่</option>
                  <option>ภูเก็ต</option>
                  <option>มหาสารคาม</option>
                  <option>มุกดาหาร</option>
                  <option>แม่ฮ่องสอน</option>
                  <option>ยโสธร</option>
                  <option>ยะลา</option>
                  <option>ร้อยเอ็ด</option>
                  <option>ระนอง</option>
                  <option>ระยอง</option>
                  <option>ราชบุรี</option>
                  <option>ลพบุรี</option>
                  <option>ลำปาง</option>
                  <option>ลำพูน</option>
                  <option>เลย</option>
                  <option>ศรีสะเกษ</option>
                  <option>สกลนคร</option>
                  <option>สงขลา</option>
                  <option>สตูล</option>
                  <option>สมุทรปราการ</option>
                  <option>สมุทรสงคราม</option>
                  <option>สมุทรสาคร</option>
                  <option>สระแก้ว</option>
                  <option>สระบุรี</option>
                  <option>สิงห์บุรี</option>
                  <option>สุโขทัย</option>
                  <option>สุพรรณบุรี</option>
                  <option>สุราษฎร์ธานี</option>
                  <option>สุรินทร์</option>
                  <option>หนองคาย</option>
                  <option>หนองบัวลำภู</option>
                  <option>อ่างทอง</option>
                  <option>อำนาจเจริญ</option>
                  <option>อุดรธานี</option>
                  <option>อุตรดิตถ์</option>
                  <option>อุทัยธานี</option>
                  <option>อุบลราชธานี</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-semibold mb-2 text-on-surface-variant">รหัสไปรษณีย์</label>
                <input type="text" inputMode="numeric" name="postal_code" value={address.postal_code} onChange={handleAddressChange}
                  maxLength={5}
                  className="w-full bg-surface-container-highest border-none rounded-lg px-4 py-4 focus:ring-2 focus:ring-primary/20 transition-all outline-none" placeholder="10xxx" />
              </div>
            </div>
          </section>

          {/* Payment Method */}
          <section className="bg-surface-container-lowest p-7 rounded-2xl">
            <div className="flex items-center gap-3 mb-6">
              <span className="material-symbols-outlined text-primary text-[22px]" style={{ fontVariationSettings: "'FILL' 1" }}>payments</span>
              <h2 className="text-lg font-bold text-on-surface">เลือกช่องทางการชำระเงิน</h2>
            </div>
            <div className="space-y-3">
              {/* PromptPay */}
              <button onClick={() => setPayment("promptpay")}
                className={`w-full p-5 rounded-xl flex items-center justify-between transition-all ${payment === "promptpay" ? "border-2 border-primary bg-primary/5" : "border border-outline-variant/20 bg-surface-container-lowest hover:border-outline-variant/40"}`}>
                <div className="flex items-center gap-4">
                  <div className="w-11 h-11 flex items-center justify-center rounded-lg bg-surface-container-highest">
                    <span className="material-symbols-outlined text-primary text-2xl">qr_code_2</span>
                  </div>
                  <div className="text-left">
                    <p className="font-bold text-sm text-on-surface">พร้อมเพย์ (PromptPay)</p>
                    <p className="text-xs text-on-surface-variant">สแกน QR Code เพื่อชำระเงินทันที</p>
                  </div>
                </div>
                <span className="material-symbols-outlined text-primary" style={payment === "promptpay" ? { fontVariationSettings: "'FILL' 1" } : {}}>
                  {payment === "promptpay" ? "check_circle" : "radio_button_unchecked"}
                </span>
              </button>

              {/* Credit Card */}
              <button onClick={() => setPayment("credit")}
                className={`w-full p-5 rounded-xl flex items-center justify-between transition-all ${payment === "credit" ? "border-2 border-primary bg-primary/5" : "border border-outline-variant/20 bg-surface-container-lowest hover:border-outline-variant/40"}`}>
                <div className="flex items-center gap-4">
                  <div className="w-11 h-11 flex items-center justify-center rounded-lg bg-secondary/10">
                    <span className="material-symbols-outlined text-secondary text-2xl">credit_card</span>
                  </div>
                  <div className="text-left">
                    <p className="font-bold text-sm text-on-surface">บัตรเครดิต / บัตรเดบิต</p>
                    <div className="flex gap-1.5 mt-1">
                      <div className="px-2 py-0.5 border border-outline-variant/20 rounded text-[10px] font-bold text-on-surface-variant">Visa</div>
                      <div className="px-2 py-0.5 border border-outline-variant/20 rounded text-[10px] font-bold text-on-surface-variant">Mastercard</div>
                    </div>
                  </div>
                </div>
                <span className="material-symbols-outlined text-primary" style={payment === "credit" ? { fontVariationSettings: "'FILL' 1" } : {}}>
                  {payment === "credit" ? "check_circle" : "radio_button_unchecked"}
                </span>
              </button>
            </div>

            {/* Stripe CardElement — แสดงเฉพาะเมื่อเลือก credit card */}
            {payment === "credit" && (
              <div className="mt-5">
                <label className="block text-sm font-semibold mb-2 text-on-surface-variant">ข้อมูลบัตร</label>
                <div className="bg-surface-container-highest rounded-xl px-4 py-4 focus-within:ring-2 focus-within:ring-primary/20 transition-all">
                  <CardElement
                    onChange={(e) => setIsCardComplete(e.complete)}
                    options={{
                      style: {
                        base: {
                          fontSize: "15px",
                          color: "var(--color-on-surface, #1a1a1a)",
                          "::placeholder": { color: "var(--color-on-surface-variant, #888)" },
                        },
                        invalid: { color: "var(--color-error, #b00020)" },
                      },
                    }}
                  />
                </div>
                <p className="text-[11px] text-on-surface-variant mt-2">
                  Test card: <span className="font-mono font-semibold">4242 4242 4242 4242</span> · exp: 12/34 · CVC: 123
                </p>
              </div>
            )}
          </section>

          {/* Error */}
          {error && (
            <div className="bg-error-container/20 text-error rounded-xl px-5 py-3.5 text-sm font-medium">
              {error}
            </div>
          )}
        </div>

        {/* Right Column: Order Summary */}
        <div className="lg:col-span-4 sticky top-24 space-y-6">
          <div className="bg-surface-container-lowest p-7 rounded-2xl border border-outline-variant/10 overflow-hidden">
            <h3 className="text-lg font-bold text-on-surface mb-6">สรุปคำสั่งซื้อ</h3>
            <div className="space-y-5 mb-6">
              {items.map((item) => (
                <div key={item.variant_id} className="flex gap-3.5">
                  <div className="w-14 h-14 rounded-xl bg-surface-container-low flex-shrink-0 overflow-hidden relative">
                    {item.image_url ? (
                      <Image className="object-cover" src={item.image_url} alt={item.product_name || ""} fill sizes="56px" />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center">
                        <span className="material-symbols-outlined text-outline text-xl">image</span>
                      </div>
                    )}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-bold text-on-surface leading-tight truncate">{item.product_name}</p>
                    <p className="text-xs text-on-surface-variant mt-1">x{item.quantity}</p>
                    <p className="text-sm font-bold text-primary mt-0.5">{formatBaht((item.price || 0) * item.quantity)}</p>
                  </div>
                </div>
              ))}
            </div>
            <div className="space-y-2.5 pt-5 border-t border-outline-variant/10">
              <div className="flex justify-between text-sm text-on-surface-variant">
                <span>ยอดรวมสินค้า</span><span className="font-medium">{formatBaht(subtotal)}</span>
              </div>
              <div className="flex justify-between text-sm text-on-surface-variant">
                <span>ค่าจัดส่ง</span><span className="font-medium">{formatBaht(shipping)}</span>
              </div>
              <div className="flex justify-between items-end pt-4 mt-1 border-t border-outline-variant/10">
                <span className="font-bold text-on-surface">ยอดชำระสุทธิ</span>
                <span className="text-xl font-black text-primary">{formatBaht(total)}</span>
              </div>
            </div>
            <button
              onClick={handleSubmit}
              disabled={!canSubmit}
              className="w-full mt-7 editorial-gradient text-white py-4 rounded-xl font-bold text-base shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all active:scale-[0.98] disabled:opacity-50"
            >
              {submitting ? "กำลังดำเนินการ..." : "ยืนยันการสั่งซื้อ"}
            </button>
            <p className="text-center text-[10px] text-on-surface-variant mt-3 px-2 leading-relaxed">
              การคลิกปุ่ม &ldquo;ยืนยันการสั่งซื้อ&rdquo; แสดงว่าคุณยอมรับ <span className="underline cursor-pointer">ข้อกำหนดและเงื่อนไข</span> ของเรา
            </p>
          </div>
        </div>
      </div>
    </>
  );
}

// ─── Page wrapper: ห่อด้วย <Elements> ให้ Stripe hooks ใช้งานได้ ─────────────
export default function CheckoutPage() {
  return (
    <Elements stripe={stripePromise}>
      <CheckoutForm />
    </Elements>
  );
}
