"use client";

import { useCart } from "@/context/cart-context";
import { useState, type ReactNode } from "react";

interface AddToCartButtonProps {
  variantId: number;
  quantity?: number;
  className?: string;
  children: ReactNode;
  // meta ใช้สำหรับ guest cart — ถ้าไม่ส่งมาก็ใช้ได้ (logged-in user ไม่ต้องการ)
  meta?: { name: string; price: number; image?: string; sku?: string };
}

export default function AddToCartButton({ variantId, quantity = 1, className, children, meta }: AddToCartButtonProps) {
  const { addItem } = useCart();
  const [isAdding, setIsAdding] = useState(false);

  const handleClick = async (e: React.MouseEvent) => {
    e.preventDefault();
    setIsAdding(true);
    try {
      await addItem(variantId, quantity, meta);
    } finally {
      setIsAdding(false);
    }
  };

  return (
    <button
      disabled={isAdding}
      onClick={handleClick}
      className={className}
    >
      {isAdding ? (
        <span className="material-symbols-outlined text-[18px] animate-spin">progress_activity</span>
      ) : (
        children
      )}
    </button>
  );
}
