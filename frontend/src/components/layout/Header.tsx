import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Moon, Sun, Wifi, WifiOff, AlertCircle, Database, RefreshCw } from "lucide-react"
import { useSystemTheme } from "@/hooks/useSystemTheme"
import { ContextSelector } from "@/components/dashboard/ContextSelector"
import { useConnectionStatus } from "@/contexts/ConnectionStatusContext"
import type { ContextInfo } from "@/components/dashboard/ContextSelector"

interface HeaderProps {
  connectionStatus: "connected" | "disconnected" | "connecting"
  onContextChange?: (context: ContextInfo) => void
}

export function Header({ onContextChange }: HeaderProps) {
  const { theme, toggleTheme } = useSystemTheme()
  const { connectionStatus: detailedStatus, refreshStatus } = useConnectionStatus()
  
  const getStatusVariant = () => {
    switch (detailedStatus.status) {
      case "connected":
        return "default"
      case "disconnected":
        return "destructive"
      case "no_contexts":
        return "destructive"
      case "invalid_context":
        return "secondary"
      case "loading":
        return "secondary"
      default:
        return "secondary"
    }
  }

  const getStatusText = () => {
    switch (detailedStatus.status) {
      case "connected":
        return "Connected"
      case "disconnected":
        return "Disconnected"
      case "no_contexts":
        return "No Contexts"
      case "invalid_context":
        return "Invalid Context"
      case "loading":
        return "Checking..."
      default:
        return "Unknown"
    }
  }

  const getStatusIcon = () => {
    switch (detailedStatus.status) {
      case "connected":
        return <Wifi className="h-3 w-3 mr-1" />
      case "disconnected":
        return <WifiOff className="h-3 w-3 mr-1" />
      case "no_contexts":
        return <Database className="h-3 w-3 mr-1" />
      case "invalid_context":
        return <AlertCircle className="h-3 w-3 mr-1" />
      case "loading":
        return <RefreshCw className="h-3 w-3 mr-1 animate-spin" />
      default:
        return <AlertCircle className="h-3 w-3 mr-1" />
    }
  }

  return (
    <header className="bg-background border-b p-2">
      <div className="container mx-auto flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold text-foreground">
            KubePulse
          </h1>
          <p className="text-sm text-muted-foreground">
            Kubernetes Health Monitoring
          </p>
        </div>
        <div className="flex items-center gap-2">
          <ContextSelector onContextChange={onContextChange} />
          <div className="w-px h-4 bg-border" />
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
          >
            {theme === 'light' ? (
              <Moon className="h-5 w-5" />
            ) : (
              <Sun className="h-5 w-5" />
            )}
          </Button>
          <Badge 
            variant={getStatusVariant()} 
            className="px-2 py-1 cursor-pointer"
            onClick={() => detailedStatus.canRetry && refreshStatus()}
            title={detailedStatus.message}
          >
            {getStatusIcon()}
            {getStatusText()}
          </Badge>
        </div>
      </div>
    </header>
  )
}