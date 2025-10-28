import { Home, Play, Library, Settings } from "lucide-react";
import { LucideIcon } from "lucide-react";

export interface NavItem {
  title: string;
  href: string;
  icon: LucideIcon;
  description?: string;
}

export const navItems: NavItem[] = [
  {
    title: "Home",
    href: "/",
    icon: Home,
    description: "Dashboard and overview",
  },
  {
    title: "Channels",
    href: "/channels",
    icon: Play,
    description: "Manage TV channels",
  },
  {
    title: "Library",
    href: "/library",
    icon: Library,
    description: "Browse media files",
  },
  {
    title: "Settings",
    href: "/settings",
    icon: Settings,
    description: "Configure preferences",
  },
];

