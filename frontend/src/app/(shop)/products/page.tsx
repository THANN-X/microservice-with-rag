import { serverFetchProducts, serverFetchCategories } from "@/lib/server-fetch";
import { formatBaht, getMinPrice } from "@/lib/utils";
import Link from "next/link";
import Image from "next/image";
import AddToCartButton from "./_components/add-to-cart-button";
import { APP_CONFIG } from "@/lib/constants";

type SearchParams = {
  search?: string;
  category?: string;
  page?: string;
};

const ITEMS_PER_PAGE = APP_CONFIG.PAGINATION.SHOP_PRODUCTS;

export default async function ProductsPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const resolvedParams = await searchParams;  // ← await ก่อนใช้
  const page = Number(resolvedParams.page) || 1;
  const search = resolvedParams.search || "";
  const category = resolvedParams.category || "";

  const categories = await serverFetchCategories();

  // Helper to find category and collect all descendant IDs recursively
  const getDescendantIds = (nodes: any[], targetId: number): number[] => {
    const findCategoryNode = (treeNodes: any[], id: number): any | null => {
      for (const node of treeNodes) {
        if (node.id === id) return node;
        if (node.children && node.children.length > 0) {
          const found = findCategoryNode(node.children, id);
          if (found) return found;
        }
      }
      return null;
    };

    const collectAllIds = (node: any): number[] => {
      const ids = [node.id];
      if (node.children && node.children.length > 0) {
        for (const child of node.children) {
          ids.push(...collectAllIds(child));
        }
      }
      return ids;
    };

    const targetNode = findCategoryNode(nodes, targetId);
    if (!targetNode) return [targetId];
    return collectAllIds(targetNode);
  };

  let categoryIds: number[] | undefined;
  if (category) {
    const targetId = Number(category);
    if (!isNaN(targetId)) {
      categoryIds = getDescendantIds(categories, targetId);
    }
  }

  const productsRes = await serverFetchProducts({
    page,
    limit: ITEMS_PER_PAGE,
    search: search || undefined,
    category: category || undefined,
    categoryIds,
  });

  const products = productsRes.items ?? [];
  const total = productsRes.total ?? 0;

  // Build a URLSearchParams for pagination links, preserving search/category
  const paginationParams = (targetPage: number) => {
    const p = new URLSearchParams();
    if (search) p.set("search", search);
    if (category) p.set("category", category);
    p.set("page", String(targetPage));
    return p.toString();
  };

  return (
    <>
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-black text-on-surface tracking-tight mb-1">
          {search ? `ผลการค้นหา "${search}"` : "สินค้าทั้งหมด"}
        </h1>
        <p className="text-on-surface-variant text-sm">{products.length} สินค้า</p>
      </div>

      {/* Category Quick Filters */}
      {categories.length > 0 && (
        <div className="flex gap-3 mb-8 overflow-x-auto no-scrollbar pb-2">
          <Link
            href="/products"
            className={`px-5 py-2 rounded-full text-sm font-bold whitespace-nowrap transition-all ${
              !category ? "bg-primary text-white" : "bg-surface-container-lowest text-on-surface hover:bg-surface-container"
            }`}
          >
            ทั้งหมด
          </Link>
          {categories.map((cat) => (
            <Link
              key={cat.id}
              href={`/products?category=${cat.id}`}
              className={`px-5 py-2 rounded-full text-sm font-bold whitespace-nowrap transition-all ${
                category === String(cat.id) ? "bg-primary text-white" : "bg-surface-container-lowest text-on-surface hover:bg-surface-container"
              }`}
            >
              {cat.name}
            </Link>
          ))}
        </div>
      )}

      {/* Grid */}
      {products.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 gap-4">
          <span className="material-symbols-outlined text-6xl text-outline">search_off</span>
          <p className="text-on-surface-variant font-medium">ไม่พบสินค้าที่ค้นหา</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-5">
          {products.map((product) => (
            <div
              key={product.id}
              className="bg-surface-container-lowest rounded-2xl p-5 group hover:shadow-lg hover:-translate-y-0.5 transition-all duration-200 relative"
            >
              <Link href={`/products/${product.id}`} className="absolute inset-0 z-0" aria-label={product.name} />
              <div className="relative overflow-hidden rounded-xl aspect-[4/3] mb-4">
                {product.image_urls?.[0] ? (
                  <Image
                    alt={product.name}
                    className="object-cover group-hover:scale-105 transition-transform duration-500"
                    src={product.image_urls[0]}
                    fill
                    sizes="(max-width: 640px) 100vw, (max-width: 1024px) 33vw, 25vw"
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
                  product.variants[0].stock > 0 ? (
                    <AddToCartButton
                      variantId={product.variants[0].id}
                      meta={{ name: product.name, price: product.variants[0].price, image: product.image_urls?.[0], sku: product.variants[0].sku }}
                      className="relative z-10 w-9 h-9 rounded-lg bg-primary/5 text-primary flex items-center justify-center hover:bg-primary hover:text-white transition-colors"
                    >
                      <span className="material-symbols-outlined text-[18px]">add</span>
                    </AddToCartButton>
                  ) : (
                    <span className="relative z-10 text-[10px] font-bold text-error bg-error/8 border border-error/20 px-2 py-1 rounded-lg whitespace-nowrap">
                      หมดแล้ว
                    </span>
                  )
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {total > ITEMS_PER_PAGE && (
        <div className="flex justify-center items-center gap-2 mt-10">
          <Link
            href={page > 1 ? `/products?${paginationParams(page - 1)}` : "#"}
            aria-disabled={page <= 1}
            className={`w-10 h-10 rounded-xl bg-surface-container-lowest flex items-center justify-center font-bold transition-colors ${
              page <= 1 ? "opacity-30 pointer-events-none" : "hover:bg-surface-container-low"
            }`}
          >
            <span className="material-symbols-outlined text-[20px]">chevron_left</span>
          </Link>
          <span className="w-10 h-10 rounded-xl bg-primary text-white font-bold text-sm flex items-center justify-center">
            {page}
          </span>
          <Link
            href={page * ITEMS_PER_PAGE < total ? `/products?${paginationParams(page + 1)}` : "#"}
            aria-disabled={page * ITEMS_PER_PAGE >= total}
            className={`w-10 h-10 rounded-xl bg-surface-container-lowest flex items-center justify-center font-bold transition-colors ${
              page * ITEMS_PER_PAGE >= total ? "opacity-30 pointer-events-none" : "hover:bg-surface-container-low"
            }`}
          >
            <span className="material-symbols-outlined text-[20px]">chevron_right</span>
          </Link>
        </div>
      )}
    </>
  );
}
