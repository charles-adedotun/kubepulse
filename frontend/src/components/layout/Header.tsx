import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Moon, Sun } from "lucide-react"
import { useSystemTheme } from "@/hooks/useSystemTheme"
import { ContextSelector } from "@/components/dashboard/ContextSelector"
import type { ContextInfo } from "@/components/dashboard/ContextSelector"

interface HeaderProps {
  connectionStatus: "connected" | "disconnected" | "connecting"
  onContextChange?: (context: ContextInfo) => void
}

export function Header({ connectionStatus, onContextChange }: HeaderProps) {
  const { theme, toggleTheme } = useSystemTheme()
  const getStatusVariant = () => {
    switch (connectionStatus) {
      case "connected":
        return "default"
      case "disconnected":
        return "destructive"
      default:
        return "secondary"
    }
  }

  const getStatusText = () => {
    switch (connectionStatus) {
      case "connected":
        return "Connected"
      case "disconnected":
        return "Disconnected"
      default:
        return "Connecting..."
    }
  }

  return (
    <header className="bg-gradient-to-r from-primary/90 to-primary p-6 shadow-lg">
      <div className="container mx-auto flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-primary-foreground flex items-center gap-2">
            <span>🚀</span>
            KubePulse Dashboard
          </h1>
          <p className="text-sm text-primary-foreground/80 mt-1">
            Intelligent Kubernetes Health Monitoring with ML-powered Anomaly Detection
          </p>
        </div>
        <div className="flex items-center gap-4">
          <ContextSelector onContextChange={onContextChange} />
          <div className="w-px h-8 bg-primary-foreground/20" />
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            className="text-primary-foreground hover:bg-primary-foreground/10"
          >
            {theme === 'light' ? (
              <Moon className="h-5 w-5" />
            ) : (
              <Sun className="h-5 w-5" />
            )}
          </Button>
          <Badge variant={getStatusVariant()} className="px-3 py-1">
            {getStatusText()}
          </Badge>
        </div>
      </div>
    </header>
  )
}