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
      "transition-colors duration-200",
      className
    )}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-1">
        <CardTitle className="text-base font-semibold">{title}</CardTitle>
        <div className={cn(
          "h-2 w-2 rounded-full",
          getStatusColor()
        )} />
      </CardHeader>
      <CardContent>
        <div className="text-lg font-semibold text-foreground mb-1">{value}</div>
        <p className="text-sm text-muted-foreground">{description}</p>
      </CardContent>
    </Card>
  )
}