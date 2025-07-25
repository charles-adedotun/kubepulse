import type { ReactNode } from "react"

interface DashboardLayoutProps {
  children: ReactNode
}

export function DashboardLayout({ children }: DashboardLayoutProps) {
  return (
    <div className="min-h-screen bg-background">
      <main className="container mx-auto px-2 py-3">
        {children}
      </main>
    </div>
  )
}