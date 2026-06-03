import Navbar from "@/components/navbar";
import Sidebar from "@/components/sidebar";
import Footer from "@/components/footer";
import AIFloatingChat from "@/components/ai-floating-chat";
import MobileNav from "@/components/mobile-nav";

export default function CustomerLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <Navbar />
      <Sidebar />
      <main className="lg:ml-64 pt-20 px-5 md:px-10 pb-24">
        {children}
      </main>
      <div className="lg:ml-64">
        <Footer />
      </div>
      <AIFloatingChat />
      <MobileNav />
    </>
  );
}
