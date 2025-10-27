import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Hermes - Virtual TV Channel Service",
  description: "Manage and stream your own virtual TV channels",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>{children}</body>
    </html>
  );
}
