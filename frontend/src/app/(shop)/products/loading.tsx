export default function ProductsLoading() {
  return (
    <>
      {/* Header skeleton */}
      <div className="mb-8">
        <div className="h-9 w-48 rounded-xl bg-surface-container animate-pulse mb-2" />
        <div className="h-4 w-20 rounded-lg bg-surface-container animate-pulse" />
      </div>

      {/* Category pills skeleton */}
      <div className="flex gap-3 mb-8 overflow-x-auto no-scrollbar pb-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <div
            key={i}
            className="h-9 rounded-full bg-surface-container animate-pulse shrink-0"
            style={{ width: `${64 + i * 12}px` }}
          />
        ))}
      </div>

      {/* Product grid skeleton */}
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-5">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="bg-surface-container-lowest rounded-2xl p-5">
            <div className="rounded-xl aspect-[4/3] mb-4 bg-surface-container animate-pulse" />
            <div className="h-5 w-3/4 rounded-lg bg-surface-container animate-pulse mb-2" />
            <div className="h-3 w-full rounded-lg bg-surface-container animate-pulse mb-4" />
            <div className="flex items-center justify-between">
              <div className="h-6 w-20 rounded-lg bg-surface-container animate-pulse" />
              <div className="w-9 h-9 rounded-lg bg-surface-container animate-pulse" />
            </div>
          </div>
        ))}
      </div>
    </>
  );
}
