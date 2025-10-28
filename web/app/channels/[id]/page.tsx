import { use } from "react";
import { MainLayout } from "@/components/layout/main-layout";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { Pencil } from "lucide-react";

export default function ChannelPlayerPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  
  return (
    <MainLayout>
      <PageHeader
        title={`Channel ${id}`}
        description="Live channel player"
        actions={
          <Button asChild variant="outline">
            <Link href={`/channels/${id}/edit`}>
              <Pencil className="h-4 w-4 mr-2" />
              Edit
            </Link>
          </Button>
        }
      />
      <div className="mt-6">
        <p className="text-muted-foreground">
          Channel player will be implemented in PBI-10
        </p>
      </div>
    </MainLayout>
  );
}

