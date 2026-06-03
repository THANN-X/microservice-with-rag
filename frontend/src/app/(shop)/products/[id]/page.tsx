import { Metadata } from "next";
import { notFound } from "next/navigation";
import { serverFetchProduct } from "@/lib/server-fetch";
import Link from "next/link";
import ProductDetailClient from "./_components/product-detail-client";

type Props = { params: Promise<{ id: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const resolvedParams = await params;
  const product = await serverFetchProduct(Number(resolvedParams.id));
  
  if (!product) {
    return {
      title: "ไม่พบสินค้า | THANN",
      description: "ขออภัยไม่พบสินค้าที่คุณต้องการ",
    };
  }

  const imageUrl = product.image_urls?.[0] || "";

  return {
    title: `${product.name} | THANN`,
    description: product.description || "สั่งซื้อผลิตภัณฑ์คุณภาพจาก THANN",
    openGraph: {
      title: `${product.name} | THANN`,
      description: product.description || "สั่งซื้อผลิตภัณฑ์คุณภาพจาก THANN",
      images: imageUrl ? [{ url: imageUrl }] : [],
    },
  };
}

export default async function ProductDetailPage({ params }: Props) {
  const resolvedParams = await params;
  const product = await serverFetchProduct(Number(resolvedParams.id));
  if (!product) notFound();

  return (
    <div className="max-w-6xl mx-auto">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-on-surface-variant mb-6">
        <Link href="/" className="hover:text-primary transition-colors">หน้าหลัก</Link>
        <span className="material-symbols-outlined text-[14px]" aria-hidden="true">chevron_right</span>
        <Link href="/products" className="hover:text-primary transition-colors">สินค้า</Link>
        <span className="material-symbols-outlined text-[14px]" aria-hidden="true">chevron_right</span>
        <span className="text-on-surface font-medium truncate max-w-[200px]">{product.name}</span>
      </div>

      {/* Interactive section (image gallery, variants, quantity, add to cart) */}
      <ProductDetailClient product={product} />

      {/* Description */}
      <div className="mt-14 grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <div className="bg-surface-container-lowest p-8 rounded-2xl">
            <h2 className="text-xl font-bold mb-4 text-on-surface">รายละเอียดสินค้า</h2>
            <div className="thai-line-height text-on-surface-variant text-sm leading-relaxed">
              <p>{product.description}</p>
            </div>
          </div>
        </div>
        <div>
          <div className="editorial-gradient p-7 rounded-2xl text-white">
            <h3 className="text-lg font-bold mb-3">โปรโมชั่นประจำเดือน</h3>
            <p className="text-white/70 text-sm mb-5 thai-line-height">ซื้อคู่ลดทันที 500 บาท</p>
            <Link href="/products" className="block w-full bg-white text-primary py-2.5 rounded-xl font-bold text-sm text-center hover:shadow-lg transition-shadow">
              ดูแพ็กเกจคู่
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
