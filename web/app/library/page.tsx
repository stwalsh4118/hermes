import { MainLayout } from "@/components/layout/main-layout";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import { RefreshCw } from "lucide-react";

export default function LibraryPage() {
  return (
    <MainLayout>
      <PageHeader
        title="Media Library"
        description="Browse and manage your media files"
        actions={
          <Button variant="outline">
            <RefreshCw className="h-4 w-4 mr-2" />
            Scan Library
          </Button>
        }
      />
      <div className="mt-6">
        <p className="text-muted-foreground">
          Media library interface will be implemented in PBI-9
        </p>
      </div>
    </MainLayout>
  );
}

