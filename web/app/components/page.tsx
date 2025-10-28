"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { LoadingSpinner } from "@/components/common/loading-spinner";
import { SkeletonCard } from "@/components/common/skeleton-card";
import { SkeletonList } from "@/components/common/skeleton-list";
import { EmptyState } from "@/components/common/empty-state";
import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { InboxIcon, RefreshCw } from "lucide-react";
import { toast } from "sonner";

export default function ComponentsPage() {
  const [confirmOpen, setConfirmOpen] = useState(false);

  return (
    <div className="container mx-auto py-10 space-y-8">
      <div>
        <h1 className="text-4xl font-bold mb-2">Common Components</h1>
        <p className="text-muted-foreground">Loading, error, and empty states</p>
      </div>

      <Separator />

      <Card>
        <CardHeader>
          <CardTitle>Loading Spinner</CardTitle>
        </CardHeader>
        <CardContent className="flex gap-4 items-center">
          <div className="flex flex-col items-center gap-2">
            <LoadingSpinner size="sm" />
            <span className="text-xs text-muted-foreground">Small</span>
          </div>
          <div className="flex flex-col items-center gap-2">
            <LoadingSpinner size="md" />
            <span className="text-xs text-muted-foreground">Medium</span>
          </div>
          <div className="flex flex-col items-center gap-2">
            <LoadingSpinner size="lg" />
            <span className="text-xs text-muted-foreground">Large</span>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Skeleton Card</CardTitle>
        </CardHeader>
        <CardContent className="bg-background p-4 rounded-lg border">
          <SkeletonCard />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Skeleton List</CardTitle>
        </CardHeader>
        <CardContent className="bg-background p-4 rounded-lg border">
          <SkeletonList count={3} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Toast Notifications</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button 
            variant="outline"
            onClick={() => toast.success("Operation completed successfully!")}
          >
            Success Toast
          </Button>
          <Button 
            variant="outline"
            onClick={() => toast.error("Failed to load data from the server")}
          >
            Error Toast
          </Button>
          <Button 
            variant="outline"
            onClick={() => toast.info("New update available")}
          >
            Info Toast
          </Button>
          <Button 
            variant="outline"
            onClick={() => toast.warning("Please save your changes")}
          >
            Warning Toast
          </Button>
          <Button 
            variant="outline"
            onClick={() => toast("Simple message")}
          >
            Default Toast
          </Button>
          <Button 
            variant="outline"
            onClick={() => 
              toast.error("Failed to load data", {
                description: "Unable to connect to the server",
                action: {
                  label: "Retry",
                  onClick: () => toast.success("Retrying..."),
                },
              })
            }
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Error with Retry
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Empty State</CardTitle>
        </CardHeader>
        <CardContent className="bg-muted/20 rounded-lg">
          <EmptyState
            icon={InboxIcon}
            title="No channels yet"
            description="Get started by creating your first TV channel"
            action={{
              label: "Create Channel",
              onClick: () => toast.success("Channel creation dialog would open here"),
            }}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Confirm Dialog</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-4">
            Click the button below to test the confirmation dialog
          </p>
          <Button 
            onClick={() => setConfirmOpen(true)} 
            variant="destructive"
            className="bg-red-600 hover:bg-red-700 text-white"
          >
            Delete Action
          </Button>
          <ConfirmDialog
            open={confirmOpen}
            onOpenChange={setConfirmOpen}
            title="Are you sure?"
            description="This action cannot be undone. This will permanently delete the channel."
            confirmLabel="Delete"
            cancelLabel="Cancel"
            variant="destructive"
            onConfirm={() => {
              toast.success("Channel deleted successfully");
              setConfirmOpen(false);
            }}
          />
        </CardContent>
      </Card>
    </div>
  );
}
