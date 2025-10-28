import type { Metadata } from "next";
import { createMetadata } from "@/lib/metadata";
import { MainLayout } from "@/components/layout/main-layout";
import { PageHeader } from "@/components/layout/page-header";

export const metadata: Metadata = createMetadata({
  title: "Settings",
  description: "Configure your service preferences",
  path: "/settings",
});

export default function SettingsPage() {
  return (
    <MainLayout>
      <PageHeader
        title="Settings"
        description="Configure your service preferences"
      />
      <div className="mt-6">
        <p className="text-muted-foreground">
          Settings interface will be implemented in PBI-11
        </p>
      </div>
    </MainLayout>
  );
}

