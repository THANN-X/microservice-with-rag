export default function CouponsPage() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] gap-4">
      <div className="w-20 h-20 rounded-full bg-surface-container-low flex items-center justify-center">
        <span className="material-symbols-outlined text-5xl text-outline">confirmation_number</span>
      </div>
      <h1 className="text-2xl font-black text-on-surface">คูปองของคุณ</h1>
      <p className="text-sm text-on-surface-variant">ยังไม่มีคูปองที่พร้อมใช้งาน</p>
    </div>
  );
}
