import { ThemeToggle } from "@/components/common/theme-toggle";

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-24">
      <div className="absolute top-4 right-4">
        <ThemeToggle />
      </div>
      <h1 className="text-4xl font-bold">Hermes</h1>
      <p className="mt-4 text-lg text-muted-foreground">
        Virtual TV Channel Service
      </p>
      <div className="mt-8 p-4 rounded-lg border bg-card text-card-foreground">
        <p>Theme toggle is working! Try switching themes.</p>
      </div>
    </main>
  );
}
