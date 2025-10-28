import type { Metadata } from "next";

interface PageMetadata {
  title: string;
  description: string;
  path?: string;
}

export function createMetadata({ title, description, path = "" }: PageMetadata): Metadata {
  const appUrl = process.env.NEXT_PUBLIC_APP_URL || "http://localhost:3000";
  const url = `${appUrl}${path}`;

  return {
    title,
    description,
    openGraph: {
      title,
      description,
      url,
    },
    twitter: {
      card: "summary_large_image",
      title,
      description,
    },
  };
}

