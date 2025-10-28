import { MainLayout } from "@/components/layout/main-layout";
import { PageHeader } from "@/components/layout/page-header";

export default function EditChannelPage({
  params,
}: {
  params: { id: string };
}) {
  return (
    <MainLayout>
      <PageHeader
        title={`Edit Channel ${params.id}`}
        description="Update channel settings and playlist"
      />
      <div className="mt-6">
        <p className="text-muted-foreground">
          Channel editor will be implemented in PBI-8
        </p>
      </div>
    </MainLayout>
  );
}

