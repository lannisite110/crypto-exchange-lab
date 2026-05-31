import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Explorer — Crypto Exchange Lab",
  description: "Blockchain explorer (Phase 5)",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
