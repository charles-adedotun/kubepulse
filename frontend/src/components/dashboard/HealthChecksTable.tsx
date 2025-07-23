import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

export interface HealthCheck {
  name: string
  status: "healthy" | "degraded" | "unhealthy"
  message: string
}

interface HealthChecksTableProps {
  checks: HealthCheck[]
}

export function HealthChecksTable({ checks }: HealthChecksTableProps) {
  const getStatusVariant = (status: HealthCheck["status"]) => {
    switch (status) {
      case "healthy":
        return "default"
      case "degraded":
        return "secondary"
      case "unhealthy":
        return "destructive"
    }
  }

  const getStatusBorderColor = (status: HealthCheck["status"]) => {
    switch (status) {
      case "healthy":
        return "border-l-green-500"
      case "degraded":
        return "border-l-yellow-500"
      case "unhealthy":
        return "border-l-red-500"
    }
  }

  if (checks.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>🔍</span>
            Health Checks
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            No health checks available
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="bg-card/50 backdrop-blur-sm border-border/50 shadow-lg dark:shadow-black/20">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-lg">
          <span>🔍</span>
          Health Checks
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-3">
          {checks.map((check, index) => (
            <div
              key={index}
              className={cn(
                "relative overflow-hidden rounded-lg p-4 border-l-4",
                "bg-secondary/30 hover:bg-secondary/50",
                "transition-all duration-200 hover:shadow-md",
                "animate-in animate-slide-in",
                getStatusBorderColor(check.status)
              )}
              style={{ animationDelay: `${index * 50}ms` }}
            >
              <div className="absolute inset-0 bg-gradient-to-r from-transparent to-background/5" />
              <div className="relative flex items-start justify-between">
                <div className="flex-1">
                  <h3 className="font-semibold text-base mb-1 text-foreground/90">{check.name}</h3>
                  <p className="text-sm text-muted-foreground">{check.message}</p>
                </div>
                <Badge 
                  variant={getStatusVariant(check.status)} 
                  className={cn(
                    "ml-4 font-semibold",
                    check.status === "healthy" && "bg-green-500/10 text-green-600 border-green-500/20",
                    check.status === "degraded" && "bg-yellow-500/10 text-yellow-600 border-yellow-500/20",
                    check.status === "unhealthy" && "bg-red-500/10 text-red-600 border-red-500/20"
                  )}
                >
                  {check.status}
                </Badge>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}