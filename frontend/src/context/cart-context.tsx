"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useRef,
  type ReactNode,
} from "react";
import type { Cart, CartItem } from "@/lib/types";
import { cartService } from "@/lib/services";
import { useAuth } from "./auth-context";
import { useToast } from "./toast-context";

// What: payload object สำหรับ updateItem — เพิ่ม field ใหม่ได้โดยไม่ต้องแก้ signature
export interface UpdateItemPayload {
  variantId: number;
  quantity: number;
  note?: string; // reserved สำหรับอนาคต เช่น หมายเหตุสินค้า
}

// What: Guest cart item เก็บใน localStorage สำหรับผู้ใช้ที่ยังไม่ได้ login
interface GuestCartItem {
  variantId: number;
  quantity: number;
  name: string;
  price: number;
  image?: string;
  sku?: string;
}

const GUEST_CART_KEY = "guest_cart";

function loadGuestCart(): GuestCartItem[] {
  try {
    const raw = localStorage.getItem(GUEST_CART_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveGuestCart(items: GuestCartItem[]) {
  localStorage.setItem(GUEST_CART_KEY, JSON.stringify(items));
}

// แปลง GuestCartItem[] เป็น Cart shape เพื่อให้ UI ใช้ร่วมกันได้
function guestItemsToCart(items: GuestCartItem[]): Cart {
  return {
    cart_id: 0,
    user_id: 0,
    items: items.map((g) => ({
      variant_id: g.variantId,
      quantity: g.quantity,
      price: g.price,
      product_name: g.name,
      variant_name: g.sku ?? "",
      image_url: g.image ?? "",
    } as CartItem)),
    created_at: "",
    updated_at: "",
  };
}

interface CartState {
  cart: Cart | null;
  itemCount: number;
  loading: boolean;
  addingItem: boolean; // #1 loading state สำหรับ addItem
  isGuest: boolean;    // #2 true เมื่อ user ยังไม่ login
  // What: Set ของ variantId ที่กำลัง pending อยู่ — ใช้แสดง spinner ระดับ item
  updatingItems: Set<number>;
  refresh: () => Promise<void>;
  addItem: (variantId: number, quantity: number, meta?: { name: string; price: number; image?: string; sku?: string }) => Promise<void>;
  updateItem: (payload: UpdateItemPayload) => void;
  removeItem: (variantId: number) => Promise<void>;
  clearCart: () => Promise<void>;
}

const CartContext = createContext<CartState | undefined>(undefined);

export function CartProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const toast = useToast();
  const [cart, setCart] = useState<Cart | null>(null);
  const [loading, setLoading] = useState(false);
  const [addingItem, setAddingItem] = useState(false); // #1
  // What: ใช้ Set เพราะ lookup O(1) และ immutable spread ทำให้ React re-render ถูกต้อง
  const [updatingItems, setUpdatingItems] = useState<Set<number>>(new Set());

  const isGuest = !user; // #2

  const setItemUpdating = (variantId: number, isUpdating: boolean) => {
    setUpdatingItems((prev) => {
      const next = new Set(prev);
      isUpdating ? next.add(variantId) : next.delete(variantId);
      return next;
    });
  };

  // What: เก็บ debounce timer แยกต่อ variantId เพื่อไม่ให้การกดสินค้าชิ้นหนึ่ง
  //       ยกเลิก timer ของสินค้าอีกชิ้น
  const debounceTimers = useRef<Map<number, ReturnType<typeof setTimeout>>>(new Map());

  // #3 Cleanup: ล้าง timer ทั้งหมดเมื่อ component unmount
  useEffect(() => {
    return () => {
      debounceTimers.current.forEach((t) => clearTimeout(t));
      debounceTimers.current.clear();
    };
  }, []);

  // #6 Cross-tab sync: ฟัง storage event จาก tab อื่น
  // Why: storage event ไม่ fire ใน tab ที่เขียนเอง — จึงใช้ sync ข้าม tab ได้พอดี
  useEffect(() => {
    if (!isGuest) return;
    const handleStorage = (e: StorageEvent) => {
      if (e.key !== GUEST_CART_KEY) return;
      try {
        const items: GuestCartItem[] = e.newValue ? JSON.parse(e.newValue) : [];
        setCart(items.length > 0 ? guestItemsToCart(items) : null);
      } catch {
        // JSON parse fail — ปล่อยผ่าน
      }
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, [isGuest]);

  const refresh = useCallback(async () => {
    if (!user) {
      // #2 โหลด guest cart จาก localStorage แทน
      const items = loadGuestCart();
      setCart(items.length > 0 ? guestItemsToCart(items) : null);
      return;
    }

    // #5 Merge: ถ้า login แล้วมีของใน guest cart ให้โอนเข้า server cart ก่อน
    const guestItems = loadGuestCart();
    if (guestItems.length > 0) {
      try {
        // ส่งทีละ item — cartService.addItem จะ merge quantity ถ้า variant ซ้ำอยู่แล้ว
        for (const item of guestItems) {
          await cartService.addItem({
            variant_id: item.variantId,
            quantity: item.quantity,
            product_name: item.name,
            variant_name: item.sku,
            price: item.price,
            image_url: item.image,
          });
        }
      } catch {
        // merge บางชิ้นอาจ fail (เช่น variant ถูกลบ) — ไม่ต้อง block
      } finally {
        localStorage.removeItem(GUEST_CART_KEY);
      }
    }

    setLoading(true);
    try {
      const c = await cartService.get();
      setCart(c);
    } catch {
      setCart(null);
    } finally {
      setLoading(false);
    }
  }, [user]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  // #1 addItem: loading state + error handling + guest support
  const addItem = async (
    variantId: number,
    quantity: number,
    meta?: { name: string; price: number; image?: string; sku?: string }
  ) => {
    if (isGuest) {
      // #2 Guest cart — เก็บลง localStorage
      const items = loadGuestCart();
      const existing = items.find((i) => i.variantId === variantId);
      if (existing) {
        existing.quantity += quantity;
      } else {
        items.push({
          variantId,
          quantity,
          name: meta?.name ?? "",
          price: meta?.price ?? 0,
          image: meta?.image,
          sku: meta?.sku,
        });
      }
      saveGuestCart(items);
      setCart(guestItemsToCart(items));
      toast("เพิ่มสินค้าลงตะกร้าแล้ว", "success");
      return;
    }

    setAddingItem(true);
    try {
      const c = await cartService.addItem({
        variant_id: variantId,
        quantity,
        product_name: meta?.name,
        variant_name: meta?.sku,
        price: meta?.price,
        image_url: meta?.image,
      });
      setCart(c);
      toast("เพิ่มสินค้าลงตะกร้าแล้ว", "success");
    } catch {
      toast("ไม่สามารถเพิ่มสินค้าได้ กรุณาลองใหม่อีกครั้ง", "error");
    } finally {
      setAddingItem(false);
    }
  };

  const updateItem = ({ variantId, quantity }: UpdateItemPayload) => {
    if (isGuest) {
      // #2 Guest cart update
      const items = loadGuestCart().map((i) =>
        i.variantId === variantId ? { ...i, quantity } : i
      );
      saveGuestCart(items);
      setCart(guestItemsToCart(items));
      return;
    }

    // What: Optimistic UI — อัปเดต state ทันทีให้ผู้ใช้เห็นก่อน ไม่ต้องรอ API
    setCart((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        items: prev.items.map((item) =>
          item.variant_id === variantId ? { ...item, quantity } : item
        ),
      };
    });

    // What: Debounce — ยกเลิก timer เก่าของ variantId นี้ก่อน แล้วตั้งอันใหม่
    // Why:  ถ้ากด + หลายครั้งติดกันจะยิง API แค่ครั้งเดียวหลังหยุดกด 400ms
    const existing = debounceTimers.current.get(variantId);
    if (existing) {
      clearTimeout(existing);
      debounceTimers.current.delete(variantId); // #4 ลบของเก่าออกก่อนตั้งอันใหม่
    }

    const timer = setTimeout(async () => {
      debounceTimers.current.delete(variantId);
      setItemUpdating(variantId, true);
      try {
        const c = await cartService.updateItem(variantId, { quantity });
        setCart(c);
      } catch {
        // What: ถ้า API fail ให้ sync กลับมาจาก server เพื่อแก้ค่าที่ optimistic ไว้
        await refresh();
        // #4 แจ้ง user ว่าทำไมตัวเลขถึงเด้งกลับ
        toast("อัปเดตจำนวนสินค้าไม่สำเร็จ กรุณาลองใหม่", "error");
      } finally {
        setItemUpdating(variantId, false);
      }
    }, 400);

    debounceTimers.current.set(variantId, timer);
  };

  const removeItem = async (variantId: number) => {
    if (isGuest) {
      // #2 Guest cart remove
      const items = loadGuestCart().filter((i) => i.variantId !== variantId);
      saveGuestCart(items);
      setCart(items.length > 0 ? guestItemsToCart(items) : null);
      return;
    }

    setItemUpdating(variantId, true);
    try {
      const c = await cartService.removeItem(variantId);
      setCart(c);
    } finally {
      setItemUpdating(variantId, false);
    }
  };

  const clearCart = async () => {
    if (isGuest) {
      localStorage.removeItem(GUEST_CART_KEY);
      setCart(null);
      return;
    }
    await cartService.clear();
    setCart(null);
  };

  const itemCount = cart?.items?.reduce((n, i) => n + i.quantity, 0) ?? 0;

  return (
    <CartContext.Provider
      value={{ cart, itemCount, loading, addingItem, isGuest, updatingItems, refresh, addItem, updateItem, removeItem, clearCart }}
    >
      {children}
    </CartContext.Provider>
  );
}

export function useCart() {
  const ctx = useContext(CartContext);
  if (!ctx) throw new Error("useCart must be inside CartProvider");
  return ctx;
}
