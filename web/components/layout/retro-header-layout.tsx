import Link from "next/link"
import { Radio, Settings } from "lucide-react"
import { ThemeToggle } from "@/components/theme-toggle"

interface RetroHeaderLayoutProps {
  children: React.ReactNode
}

export function RetroHeaderLayout({ children }: RetroHeaderLayoutProps) {
  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b-4 border-primary backdrop-blur-sm sticky top-0 z-50 bg-card shadow-lg">
        <div className="container mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Link href="/" className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-lg bg-primary border-4 border-primary/30 flex items-center justify-center shadow-lg">
                  <Radio className="w-7 h-7 text-primary-foreground" />
                </div>
                <h1 className="text-2xl font-bold text-foreground vcr-text">Virtual TV</h1>
              </Link>
            </div>
            <nav className="flex items-center gap-4">
              <Link href="/channels">
                <button className="retro-button bg-muted text-foreground hover:bg-muted/60 px-4 py-2 rounded-lg font-bold text-sm border-2 border-primary/30 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all">
                  CHANNELS
                </button>
              </Link>
              <Link href="/library">
                <button className="retro-button bg-muted text-foreground hover:bg-muted/60 px-4 py-2 rounded-lg font-bold text-sm border-2 border-primary/30 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all">
                  LIBRARY
                </button>
              </Link>
              <ThemeToggle />
              <Link href="/settings">
                <button className="retro-button bg-muted text-foreground hover:bg-muted/60 p-3 rounded-lg border-2 border-primary/30 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all">
                  <Settings className="w-5 h-5" />
                </button>
              </Link>
            </nav>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-6 py-8">
        {children}
      </main>
    </div>
  )
}

