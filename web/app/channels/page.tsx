import { MainLayout } from "@/components/layout/main-layout";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { Plus } from "lucide-react";

export default function ChannelsPage() {
  return (
    <MainLayout>
      <PageHeader
        title="Channels"
        description="Manage your virtual TV channels"
        actions={
          <Button asChild>
            <Link href="/channels/new">
              <Plus className="h-4 w-4 mr-2" />
              New Channel
            </Link>
          </Button>
        }
      />
      <div className="mt-6">
        <p className="text-muted-foreground">Channel list will be implemented in PBI-8</p>
      </div>
    </MainLayout>
  );
}

