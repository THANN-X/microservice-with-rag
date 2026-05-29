import type { Metadata } from "next";
import "./globals.css";
import { AuthProvider } from "@/context/auth-context";
import { CartProvider } from "@/context/cart-context";
import GoogleOAuthWrapper from "@/components/google-oauth-wrapper";

export const metadata: Metadata = {
  title: "อนันตา — ประสบการณ์การช้อปปิ้งที่เหนือระดับ",
  description:
    "แพลตฟอร์มอีคอมเมิร์ซระดับพรีเมียม สัมผัสประสบการณ์ช้อปปิ้งที่เหนือระดับ",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="th">
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Be+Vietnam+Pro:wght@300;400;500;600;700;800;900&display=swap"
          rel="stylesheet"
        />
        <link
          href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@20..48,100..700,0..1,-50..200&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="min-h-screen bg-surface font-body text-on-surface antialiased">
        <GoogleOAuthWrapper>
          <AuthProvider>
            <CartProvider>
              {children}
            </CartProvider>
          </AuthProvider>
        </GoogleOAuthWrapper>
      </body>
    </html>
  );
}
