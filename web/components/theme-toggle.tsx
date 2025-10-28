"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Palette } from "lucide-react"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"

const themes = [
  { id: "modern", name: "Modern Clean", classes: "dark" },
  { id: "neon-arcade", name: "Neon Arcade", classes: "dark retro neon-arcade" },
  { id: "sunset-vhs", name: "Sunset VHS", classes: "dark retro sunset-vhs" },
  { id: "retro-gaming", name: "Retro Gaming", classes: "dark retro retro-gaming" },
  { id: "miami-vice", name: "Miami Vice", classes: "dark retro miami-vice" },
]

export function ThemeToggle() {
  const [currentTheme, setCurrentTheme] = useState("sunset-vhs")

  const handleThemeChange = (themeId: string) => {
    const theme = themes.find((t) => t.id === themeId)
    if (theme) {
      setCurrentTheme(themeId)
      document.documentElement.className = theme.classes
    }
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="border-2 border-primary">
          <Palette className="w-5 h-5" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        {themes.map((theme) => (
          <DropdownMenuItem
            key={theme.id}
            onClick={() => handleThemeChange(theme.id)}
            className={currentTheme === theme.id ? "bg-primary/10 font-bold" : ""}
          >
            {theme.name}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

