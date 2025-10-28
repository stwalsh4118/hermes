import { MainLayout } from "@/components/layout/main-layout";
import { PageHeader } from "@/components/layout/page-header";

export default function NewChannelPage() {
  return (
    <MainLayout>
      <PageHeader
        title="Create New Channel"
        description="Set up a new virtual TV channel"
      />
      <div className="mt-6">
        <p className="text-muted-foreground">
          Channel creation form will be implemented in PBI-8
        </p>
      </div>
    </MainLayout>
  );
}

