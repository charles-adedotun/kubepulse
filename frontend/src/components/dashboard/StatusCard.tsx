import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

interface StatusCardProps {
  title: string
  value: string | number
  description: string
  status?: "healthy" | "degraded" | "unhealthy" | "unknown"
  className?: string
}

export function StatusCard({ 
  title, 
  value, 
  description, 
  status = "unknown",
  className 
}: StatusCardProps) {
  const getStatusColor = () => {
    switch (status) {
      case "healthy":
        return "bg-green-500"
      case "degraded":
        return "bg-yellow-500"
      case "unhealthy":
        return "bg-red-500"
      default:
        return "bg-gray-500"
    }
  }

  return (
    <Card className={cn(
      "relative overflow-hidden transition-all duration-200",
      "hover:shadow-xl hover:-translate-y-0.5",
      "bg-card/50 backdrop-blur-sm border-border/50",
      "dark:shadow-lg dark:shadow-black/20",
      className
    )}>
      <div className="absolute inset-0 bg-gradient-to-br from-transparent to-primary/5 dark:to-primary/10" />
      <CardHeader className="relative flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-base font-semibold text-foreground/90">{title}</CardTitle>
        <div className={cn(
          "h-3 w-3 rounded-full shadow-sm",
          "ring-2 ring-background/50",
          getStatusColor()
        )} />
      </CardHeader>
      <CardContent className="relative">
        <div className="text-2xl font-bold text-foreground mb-1 tracking-tight">{value}</div>
        <p className="text-sm text-muted-foreground">{description}</p>
      </CardContent>
    </Card>
  )
}