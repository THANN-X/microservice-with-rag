import { serverFetchProducts, serverFetchCategories } from "@/lib/server-fetch";
import { formatBaht, getMinPrice } from "@/lib/utils";
import Link from "next/link";
import Image from "next/image";
import AddToCartButton from "./products/_components/add-to-cart-button";
import { APP_CONFIG } from "@/lib/constants";

const categoryIcons: Record<string, string> = {
  default: "category",
  "บ้านและสวน": "home",
  "แฟชั่น": "checkroom",
  "อิเล็กทรอนิกส์": "laptop_mac",
  "แกดเจ็ต": "devices_other",
  "เครื่องครัว": "flatware",
};

/* Static color maps so Tailwind can purge safely — ต้องเป็น string เต็ม */
const categoryStyles = [
  { bg: "bg-primary",     text: "text-primary",     hover: "group-hover:bg-primary" },
  { bg: "bg-secondary",   text: "text-secondary",   hover: "group-hover:bg-secondary" },
  { bg: "bg-tertiary",    text: "text-tertiary",     hover: "group-hover:bg-tertiary" },
  { bg: "bg-primary-dim", text: "text-primary-dim", hover: "group-hover:bg-primary-dim" },
  { bg: "bg-on-surface",  text: "text-on-surface",  hover: "group-hover:bg-on-surface" },
];

export default async function HomePage() {
  const [productsRes, categories] = await Promise.all([
    serverFetchProducts({ page: 1, limit: APP_CONFIG.PAGINATION.SHOP_HOME_FEATURED }),
    serverFetchCategories(),
  ]);

  const products = productsRes.items ?? [];
  const featured = products[0];
  const regularProducts = products.slice(1, 5);

  const renderCategoryCard = (name: string, icon: string, idx: number, href: string) => {
    const style = categoryStyles[idx % 5];
    return (
    <Link
      key={name}
      href={href}
      className="bg-surface-container-lowest p-6 rounded-2xl flex flex-col items-center gap-4 hover:shadow-lg hover:-translate-y-1 transition-all duration-300 cursor-pointer group"
    >
      <div className={`w-14 h-14 rounded-2xl bg-surface-container flex items-center justify-center ${style.hover} transition-colors`}>
        <span className={`material-symbols-outlined text-2xl ${style.text} group-hover:text-white`}>{icon}</span>
      </div>
      <span className="font-semibold text-sm text-on-surface">{name}</span>
    </Link>
    );
  };

  return (
    <>
      {/* Hero Banner */}
      <section className="relative mb-14 overflow-hidden rounded-2xl editorial-gradient min-h-[420px] flex items-center">
        <div className="absolute inset-0 bg-gradient-to-r from-black/10 via-transparent to-transparent" />
        <div className="relative z-10 max-w-xl px-10 md:px-16 py-14">
          <span className="inline-flex items-center gap-1.5 px-3 py-1 bg-white/20 backdrop-blur-sm text-white rounded-full text-xs font-bold mb-5">
            <span className="w-1.5 h-1.5 rounded-full bg-white animate-pulse" />
            คอลเลกชันใหม่
          </span>
          <h1 className="text-4xl md:text-6xl font-black text-white mb-5 tracking-tight leading-[1.15]">
            สัมผัสสุนทรียภาพ<br />
            <span className="text-primary-container italic">แห่งสไตล์</span>
          </h1>
          <p className="text-base text-white/80 mb-8 max-w-sm thai-line-height leading-relaxed">
            ยกระดับประสบการณ์การช้อปปิ้งด้วยสินค้าคุณภาพ ดีไซน์ที่ผสมผสานความทันสมัยและความประณีต
          </p>
          <div className="flex gap-3">
            <Link
              href="/products"
              className="bg-white text-primary px-7 py-3.5 rounded-full font-bold text-sm shadow-xl hover:shadow-2xl hover:scale-[1.03] transition-all flex items-center gap-2"
            >
              ช้อปเลย <span className="material-symbols-outlined text-[18px]">arrow_forward</span>
            </Link>
            <Link
              href="/chat"
              className="bg-white/15 backdrop-blur-sm text-white px-7 py-3.5 rounded-full font-bold text-sm border border-white/20 hover:bg-white/25 transition-colors"
            >
              ถาม AI
            </Link>
          </div>
        </div>
      </section>

      {/* Category Filters */}
      <section className="mb-14">
        <div className="flex justify-between items-end mb-6">
          <div>
            <h2 className="text-2xl font-black text-on-surface tracking-tight mb-1">เลือกตามหมวดหมู่</h2>
            <p className="text-on-surface-variant text-sm">ค้นหาสิ่งที่ใช่สำหรับสไตล์ของคุณ</p>
          </div>
          <Link href="/products" className="text-primary font-bold text-sm flex items-center gap-1 hover:underline">
            ดูทั้งหมด <span className="material-symbols-outlined text-[16px]">arrow_forward</span>
          </Link>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-4">
          {categories.length > 0
            ? categories.slice(0, 5).map((cat, idx) => {
                const icon = categoryIcons[cat.name] || categoryIcons.default;
                return renderCategoryCard(cat.name, icon, idx, `/products?category=${cat.id}`);
              })
            : ["บ้านและสวน", "แฟชั่น", "อิเล็กทรอนิกส์", "แกดเจ็ต", "เครื่องครัว"].map((name, idx) => {
                const icons = ["home", "checkroom", "laptop_mac", "devices_other", "flatware"];
                return renderCategoryCard(name, icons[idx], idx, "/products");
              })}
        </div>
      </section>

      {/* Product Grid: Bento Style */}
      <section className="mb-16">
        <h2 className="text-2xl font-black text-on-surface mb-8 tracking-tight">สินค้าแนะนำสำหรับคุณ</h2>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          {/* Large Featured Card */}
          {featured && (
            <div className="md:col-span-2 md:row-span-2 bg-surface-container-lowest rounded-2xl p-8 relative overflow-hidden group hover:shadow-xl transition-shadow">
              <Link href={`/products/${featured.id}`} className="absolute inset-0 z-0" aria-label={featured.name} />
              {featured.image_urls?.[0] && (
                <div className="w-full h-72 relative">
                  <Image
                    alt={featured.name}
                    className="object-contain group-hover:scale-105 transition-transform duration-500"
                    src={featured.image_urls[0]}
                    fill
                    sizes="(max-width: 768px) 100vw, 50vw"
                  />
                </div>
              )}
              {!featured.image_urls?.[0] && (
                <div className="w-full h-72 flex items-center justify-center bg-surface-container rounded-xl">
                  <span className="material-symbols-outlined text-7xl text-outline">image</span>
                </div>
              )}
              <div className="mt-6">
                <span className="text-tertiary font-bold text-xs tracking-widest uppercase">BEST SELLER</span>
                <h3 className="text-3xl font-black mt-2 mb-3 leading-tight">{featured.name}</h3>
                <p className="text-on-surface-variant text-sm mb-5 line-clamp-2 thai-line-height">{featured.description}</p>
                <div className="flex items-center justify-between">
                  <span className="text-2xl font-black text-primary">{formatBaht(getMinPrice(featured.variants))}</span>
                  {featured.variants?.[0] && (
                    <AddToCartButton
                      variantId={featured.variants[0].id}
                      meta={{ name: featured.name, price: featured.variants[0].price, image: featured.image_urls?.[0], sku: featured.variants[0].sku }}
                      className="relative z-10 w-12 h-12 rounded-xl bg-primary text-white flex items-center justify-center hover:bg-primary-dim transition-colors"
                    >
                      <span className="material-symbols-outlined text-[20px]">add_shopping_cart</span>
                    </AddToCartButton>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Regular Cards */}
          {regularProducts.map((product) => (
            <div key={product.id} className="bg-surface-container-lowest rounded-2xl p-5 group hover:shadow-lg transition-shadow relative">
              <Link href={`/products/${product.id}`} className="absolute inset-0 z-0" aria-label={product.name} />
              <div className="relative overflow-hidden rounded-xl aspect-[4/3] mb-4">
                {product.image_urls?.[0] ? (
                  <Image
                    alt={product.name}
                    className="object-cover group-hover:scale-105 transition-transform duration-500"
                    src={product.image_urls[0]}
                    fill
                    sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 25vw"
                  />
                ) : (
                  <div className="w-full h-full bg-surface-container flex items-center justify-center absolute inset-0">
                    <span className="material-symbols-outlined text-5xl text-outline">image</span>
                  </div>
                )}
              </div>
              <h3 className="text-base font-bold mb-1 line-clamp-1">{product.name}</h3>
              <p className="text-on-surface-variant text-xs mb-3 line-clamp-1">{product.description}</p>
              <div className="flex items-center justify-between">
                <span className="text-lg font-black text-primary">{formatBaht(getMinPrice(product.variants))}</span>
                {product.variants?.[0] && (
                  <AddToCartButton
                    variantId={product.variants[0].id}
                    meta={{ name: product.name, price: product.variants[0].price, image: product.image_urls?.[0], sku: product.variants[0].sku }}
                    className="relative z-10 w-9 h-9 rounded-lg bg-primary/5 text-primary flex items-center justify-center hover:bg-primary hover:text-white transition-colors"
                  >
                    <span className="material-symbols-outlined text-[18px]">add</span>
                  </AddToCartButton>
                )}
              </div>
            </div>
          ))}

          {/* Hot Deal Banner (if enough products) */}
          {products.length > 5 && (
            <div className="editorial-gradient rounded-2xl p-6 text-white group md:col-span-2 overflow-hidden">
              <div className="flex gap-6 items-center">
                <div className="w-1/2">
                  <span className="inline-block px-3 py-1 bg-white/20 backdrop-blur-sm rounded-full text-[10px] font-bold mb-3">HOT DEAL</span>
                  <h3 className="text-xl font-black mb-2 leading-tight">{products[5].name}</h3>
                  <p className="text-white/70 text-xs mb-4 line-clamp-2 thai-line-height">{products[5].description}</p>
                  <span className="text-xl font-black block mb-4">{formatBaht(getMinPrice(products[5].variants))}</span>
                  <Link href={`/products/${products[5].id}`} className="inline-block bg-white text-primary px-5 py-2 rounded-full font-bold text-xs hover:shadow-lg transition-shadow">
                    ดูสินค้า
                  </Link>
                </div>
                <div className="w-1/2 overflow-hidden rounded-xl relative h-40">
                  {products[5].image_urls?.[0] ? (
                    <Image
                      alt={products[5].name}
                      className="object-cover group-hover:scale-105 transition-transform duration-500"
                      src={products[5].image_urls[0]}
                      fill
                      sizes="(max-width: 768px) 50vw, 25vw"
                    />
                  ) : (
                    <div className="w-full h-full bg-white/10 flex items-center justify-center absolute inset-0">
                      <span className="material-symbols-outlined text-5xl text-white/40">image</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      </section>
    </>
  );
}
