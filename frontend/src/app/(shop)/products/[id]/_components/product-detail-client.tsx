"use client";

import { useState, useEffect } from "react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import type { Product, Variant } from "@/lib/types";
import { formatBaht } from "@/lib/utils";
import { useCart } from "@/context/cart-context";

interface ProductDetailClientProps {
  product: Product;
}

export default function ProductDetailClient({ product }: ProductDetailClientProps) {
  const router = useRouter();
  const images = product.image_urls?.length ? product.image_urls : [];

  // Build option groups from all variants
  const optionGroups: Record<string, string[]> = {};
  product.variants?.forEach((v) => {
    v.options?.forEach((opt) => {
      if (!optionGroups[opt.name]) optionGroups[opt.name] = [];
      if (!optionGroups[opt.name].includes(opt.value)) {
        optionGroups[opt.name].push(opt.value);
      }
    });
  });

  // Init state from first variant
  const firstVariant = product.variants?.[0] ?? null;
  const initOptions: Record<string, string> = {};
  firstVariant?.options?.forEach((o) => { initOptions[o.name] = o.value; });

  const [selectedVariant, setSelectedVariant] = useState<Variant | null>(firstVariant);
  const [selectedOptions, setSelectedOptions] = useState<Record<string, string>>(initOptions);
  const [quantity, setQuantity] = useState(1);
  const [mainImage, setMainImage] = useState(0);
  
  const { addItem, isGuest } = useCart();
  const [isAdding, setIsAdding] = useState(false);
  const [isBuying, setIsBuying] = useState(false);

  // Reset quantity to 1 every time the selected variant changes
  useEffect(() => {
    setQuantity(1);
  }, [selectedVariant?.id]);

  const price = selectedVariant?.price ?? 0;
  const inStock = selectedVariant ? selectedVariant.stock > 0 : false;
  const isLowStock = selectedVariant ? selectedVariant.stock > 0 && selectedVariant.stock <= 5 : false;

  // ตรวจสอบว่า option นี้ถ้าเลือกไปจะมี variant รองรับหรือเปล่า
  // โดยคำนึงถึง option อื่นๆ ที่เลือกอยู่ด้วย
  const isOptionAvailable = (attrName: string, val: string) => {
    const hypothetical = { ...selectedOptions, [attrName]: val };
    return product.variants.some((v) =>
      Object.entries(hypothetical).every(([attr, attrVal]) =>
        v.options.some((o) => o.name === attr && o.value === attrVal)
      )
    );
  };

  const handleSelectOption = (attrName: string, value: string) => {
    const next = { ...selectedOptions, [attrName]: value };

    // 1. ลองหาดูว่ามี Variant ที่ตรงกับทุก Option ที่เลือก
    let matched = product.variants.find((v) =>
      Object.entries(next).every(([attr, val]) =>
        v.options.some((o) => o.name === attr && o.value === val)
      )
    );

    // 2. ถ้าไม่มีแบบตรงกัน (ติด Deadlock)
    // ให้ดึง Variant แรกสุดที่มี Option ที่เพิ่งกดไปมาใช้แทน
    if (!matched) {
      matched = product.variants.find((v) =>
        v.options.some((o) => o.name === attrName && o.value === value)
      );

      // ถ้าระบบหา Variant สำรองเจอ ให้เปลี่ยน Option ทั้งหมดบนหน้าจอตาม Variant นี้เลย
      if (matched) {
        const newOptions: Record<string, string> = {};
        matched.options.forEach((o) => {
          newOptions[o.name] = o.value;
        });
        setSelectedOptions(newOptions);
        setSelectedVariant(matched);
        return; // ทำงานจบปุ๊บ ให้ออกจากฟังก์ชันเลย
      }
    }

    // 3. ถ้าตรงตั้งแต่แรก ก็อัปเดตตามปกติ
    setSelectedOptions(next);
    // ถ้าไม่มี variant ที่ตรง ให้ set เป็น null (ไม่ค้างค่าเก่า)
    setSelectedVariant(matched ?? null);
  };

  const handleAddToCart = async () => {
    if (!selectedVariant) return;
    setIsAdding(true);
    try {
      await addItem(selectedVariant.id, quantity, {
        name: product.name,
        price: selectedVariant.price,
        image: product.image_urls?.[0],
        sku: selectedVariant.sku,
      });
    } finally {
      setIsAdding(false);
    }
  };

  const handleBuyNow = async () => {
    if (!selectedVariant) return;
    setIsBuying(true);
    try {
      await addItem(selectedVariant.id, quantity, {
        name: product.name,
        price: selectedVariant.price,
        image: product.image_urls?.[0],
        sku: selectedVariant.sku,
      });
      // After successfully adding, redirect to checkout
      if (isGuest) {
        router.push("/login?callbackUrl=/checkout");
      } else {
        router.push("/checkout");
      }
    } catch {
      setIsBuying(false);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-10 items-start">
      {/* Image Gallery */}
      <div>
        <div className="relative rounded-2xl overflow-hidden bg-surface-container-lowest aspect-square mb-3">
          {images[mainImage] ? (
            <Image 
              src={images[mainImage]} 
              alt={product.name} 
              fill
              className="object-cover"
              sizes="(max-width: 1024px) 100vw, 50vw"
              priority
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center bg-surface-container absolute inset-0">
              <span className="material-symbols-outlined text-7xl text-outline">image</span>
            </div>
          )}
          {product.categories?.[0] && (
            <span className="absolute top-4 left-4 px-3 py-1 bg-primary/90 backdrop-blur-sm text-white rounded-lg text-xs font-bold">
              {product.categories[0].name}
            </span>
          )}
        </div>
        {images.length > 1 && (
          <div className="grid grid-cols-4 gap-2">
            {images.slice(0, 4).map((img, i) => (
              <button
                key={i}
                onClick={() => setMainImage(i)}
                className={`relative rounded-xl overflow-hidden aspect-square border-2 transition-all ${
                  mainImage === i ? "border-primary ring-2 ring-primary/20" : "border-transparent opacity-60 hover:opacity-100"
                }`}
              >
                <Image 
                  src={img} 
                  alt="" 
                  fill
                  className="object-cover" 
                  sizes="(max-width: 1024px) 25vw, 12vw"
                />
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Product Info */}
      <div className="space-y-5">
        <div>
          {product.categories?.[0] && (
            <span className="text-xs font-bold uppercase tracking-widest text-secondary">
              {product.categories[0].name}
            </span>
          )}
          <h1 className="text-3xl font-black text-on-surface mt-1 tracking-tight leading-tight">{product.name}</h1>
        </div>

        {/* Price */}
        <div className="flex items-end gap-3">
          {selectedVariant ? (
            <span className="text-3xl font-black text-primary">{formatBaht(price)}</span>
          ) : (
            <span className="text-3xl font-black text-outline/40">—</span>
          )}
          {selectedVariant && selectedVariant.stock === 0 && (
            <span className="text-xs font-bold text-error bg-error/10 border border-error/20 px-3 py-1 rounded-md">
              สินค้าหมด
            </span>
          )}
          {isLowStock && (
            <span className="text-xs font-medium text-amber-700 bg-amber-50 border border-amber-200 px-2 py-1 rounded-md">
              เหลือเพียง {selectedVariant!.stock} ชิ้น!
            </span>
          )}
          {selectedVariant && selectedVariant.stock > 5 && (
            <span className="text-xs font-medium text-on-surface-variant bg-surface-container px-2 py-1 rounded-md">
              เหลือ {selectedVariant.stock} ชิ้น
            </span>
          )}
          {!selectedVariant && Object.keys(selectedOptions).length > 0 && (
            <span className="text-xs font-medium text-error bg-error/5 border border-error/15 px-2 py-1 rounded-md">
              ไม่มีตัวเลือกนี้
            </span>
          )}
        </div>

        {/* Variant Options (grouped by attribute) */}
        {Object.entries(optionGroups).map(([attrName, values]) => (
          <div key={attrName}>
            <h3 className="font-bold text-on-surface mb-3">{attrName}</h3>
            <div className="flex gap-3 flex-wrap">
              {values.map((val) => {
                const available = isOptionAvailable(attrName, val);
                return (
                  <button
                    key={val}
                    onClick={() => handleSelectOption(attrName, val)}
                    className={`px-6 py-2 rounded-lg border-2 transition-all font-medium ${
                      selectedOptions[attrName] === val
                        ? "border-primary text-primary bg-primary/5 border-solid" 
                        : available
                        ? "border-outline-variant/50 hover:border-primary hover:text-primary border-solid"
                        : "border-dashed border-outline/30 text-outline/40 bg-surface-container-lowest line-through"
                    }`}
                  >
                    {val}
                  </button>
                );
              })}
            </div>
          </div>
        ))}

        {/* Variant select if no options */}
        {Object.keys(optionGroups).length === 0 && product.variants.length > 1 && (
          <div>
            <h3 className="font-bold text-on-surface mb-3">ตัวเลือก</h3>
            <div className="flex gap-3 flex-wrap">
              {product.variants.map((v) => (
                <button
                  key={v.id}
                  onClick={() => setSelectedVariant(v)}
                  className={`px-6 py-2 rounded-lg border transition-all font-medium ${
                    selectedVariant?.id === v.id
                      ? "border-primary text-primary bg-primary/5"
                      : "border-outline-variant hover:border-primary hover:text-primary"
                  }`}
                >
                  {v.name}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Quantity + Actions */}
        <div className="space-y-4 pt-2">
          <div>
            <h3 className="font-bold text-on-surface text-sm mb-2">จำนวน</h3>
            <div className="inline-flex items-center bg-surface-container-highest rounded-xl">
              <button
                onClick={() => setQuantity(Math.max(1, quantity - 1))}
                className="w-10 h-10 flex items-center justify-center text-on-surface-variant hover:text-primary transition-colors"
              >
                <span className="material-symbols-outlined text-[20px]">remove</span>
              </button>
              <span className="w-10 text-center font-bold text-sm">{quantity}</span>
              <button
                onClick={() => setQuantity(Math.min(selectedVariant?.stock ?? 99, quantity + 1))}
                disabled={!selectedVariant || quantity >= selectedVariant.stock}
                className="w-10 h-10 flex items-center justify-center text-on-surface-variant hover:text-primary transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
              >
                <span className="material-symbols-outlined text-[20px]">add</span>
              </button>
            </div>
          </div>
          <div className="flex gap-3">
            <button
              onClick={handleAddToCart}
              disabled={isAdding || isBuying || !selectedVariant || !inStock}
              className="flex-1 border-2 border-primary/20 text-primary py-3.5 rounded-xl font-bold text-sm hover:border-primary hover:bg-primary/5 transition-all flex items-center justify-center gap-2 disabled:opacity-60 disabled:cursor-not-allowed"
            >
              {isAdding ? (
                <span className="material-symbols-outlined text-[20px] animate-spin">progress_activity</span>
              ) : (
                <span className="material-symbols-outlined text-[20px]">shopping_cart</span>
              )}
              {selectedVariant && !inStock ? "สินค้าหมด" : "ใส่ตะกร้า"}
            </button>
            <button
              onClick={handleBuyNow}
              disabled={isAdding || isBuying || !selectedVariant || !inStock}
              className="flex-1 editorial-gradient text-white py-3.5 rounded-xl font-bold text-sm shadow-lg shadow-primary/15 hover:shadow-xl hover:shadow-primary/25 transition-shadow flex items-center justify-center gap-2 disabled:opacity-60 disabled:shadow-none disabled:cursor-not-allowed"
            >
              {isBuying ? (
                <span className="material-symbols-outlined text-[20px] animate-spin">progress_activity</span>
              ) : null}
              ซื้อเลย
            </button>
          </div>
        </div>

        {/* Guarantees */}
        <div className="space-y-3 pt-5 border-t border-surface-container">
          <div className="flex items-center gap-3 text-sm text-on-surface-variant">
            <span className="material-symbols-outlined text-[20px] text-primary" style={{ fontVariationSettings: "'FILL' 1" }}>verified</span>
            รับประกันคุณภาพ 2 ปีเต็ม
          </div>
          <div className="flex items-center gap-3 text-sm text-on-surface-variant">
            <span className="material-symbols-outlined text-[20px] text-primary" style={{ fontVariationSettings: "'FILL' 1" }}>local_shipping</span>
            จัดส่งฟรีทั่วประเทศ ภายใน 1-3 วัน
          </div>
          <div className="flex items-center gap-3 text-sm text-on-surface-variant">
            <span className="material-symbols-outlined text-[20px] text-primary" style={{ fontVariationSettings: "'FILL' 1" }}>sync</span>
            เปลี่ยนคืนได้ภายใน 30 วัน
          </div>
        </div>
      </div>
    </div>
  );
}
