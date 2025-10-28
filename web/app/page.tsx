import { MainLayout } from "@/components/layout/main-layout";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import Link from "next/link";
import { Play, Library, Settings } from "lucide-react";

export default function Home() {
  return (
    <MainLayout>
      <PageHeader
        title="Dashboard"
        description="Welcome to Hermes - Your Virtual TV Channel Service"
      />
      
      <div className="grid gap-6 md:grid-cols-3 mt-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Play className="h-5 w-5" />
              Channels
            </CardTitle>
            <CardDescription>
              Manage and stream your TV channels
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild className="w-full">
              <Link href="/channels">View Channels</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Library className="h-5 w-5" />
              Media Library
            </CardTitle>
            <CardDescription>
              Browse and organize your media files
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="secondary" className="w-full">
              <Link href="/library">Open Library</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Settings
            </CardTitle>
            <CardDescription>
              Configure your service settings
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild variant="outline" className="w-full">
              <Link href="/settings">Open Settings</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </MainLayout>
  );
}
