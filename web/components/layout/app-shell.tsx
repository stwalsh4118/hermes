import React from "react";
import { Header } from "./header";
import { MobileMenu } from "./mobile-menu";

interface AppShellProps {
  children: React.ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  return (
    <div className="relative min-h-screen flex flex-col">
      <Header />
      <main className="flex-1">{children}</main>
      <MobileMenu />
    </div>
  );
}

