import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { ThemeProvider } from "@/components/theme-provider";
import { QueryProvider } from "@/components/providers/query-provider";
import { Toaster } from "@/components/ui/sonner";

const geist = Geist({ subsets: ["latin"], variable: "--font-sans" });
const geistMono = Geist_Mono({ subsets: ["latin"], variable: "--font-mono" });

export const metadata: Metadata = {
  title: "Virtual TV - Your Personal Streaming Channels",
  description: "Create and watch your own virtual TV channels with continuous broadcasts from your media library",
  generator: "Hermes",
  keywords: ["tv channels", "streaming", "media server", "iptv", "virtual tv"],
  authors: [{ name: "Hermes Team" }],
  creator: "Hermes",
  openGraph: {
    type: "website",
    locale: "en_US",
    url: process.env.NEXT_PUBLIC_APP_URL || "http://localhost:3000",
    title: "Virtual TV - Your Personal Streaming Channels",
    description: "Create and watch your own virtual TV channels",
    siteName: "Virtual TV",
  },
  twitter: {
    card: "summary_large_image",
    title: "Virtual TV - Your Personal Streaming Channels",
    description: "Create and watch your own virtual TV channels",
  },
  robots: {
    index: true,
    follow: true,
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark retro sunset-vhs" suppressHydrationWarning>
      <body className={`${geist.variable} ${geistMono.variable} font-sans antialiased`}>
        <ThemeProvider
          attribute="class"
          defaultTheme="dark"
          enableSystem={false}
          disableTransitionOnChange
        >
          <QueryProvider>
            {children}
          </QueryProvider>
          <Toaster />
        </ThemeProvider>
      </body>
    </html>
  );
}
