import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import { TrendingUp, TrendingDown, Brain, AlertTriangle, CheckCircle } from "lucide-react"

interface AIAnalysis {
  trend: "improving" | "stable" | "degrading" | "unknown"
  prediction: string
  confidence: number
  recommendation?: string
}

interface AIEnhancedStatusCardProps {
  title: string
  value: string | number
  description: string
  status?: "healthy" | "degraded" | "unhealthy" | "unknown"
  className?: string
  aiAnalysis?: AIAnalysis
  showAIInsights?: boolean
}

export function AIEnhancedStatusCard({ 
  title, 
  value, 
  description, 
  status = "unknown",
  className,
  aiAnalysis,
  showAIInsights = true
}: AIEnhancedStatusCardProps) {
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

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case "improving":
        return <TrendingUp className="h-3 w-3 text-green-500" />
      case "degrading":
        return <TrendingDown className="h-3 w-3 text-red-500" />
      default:
        return null
    }
  }

  const getTrendColor = (trend: string) => {
    switch (trend) {
      case "improving":
        return "text-green-600"
      case "degrading":
        return "text-red-600"
      case "stable":
        return "text-blue-600"
      default:
        return "text-gray-600"
    }
  }

  const getStatusIcon = () => {
    switch (status) {
      case "healthy":
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case "unhealthy":
        return <AlertTriangle className="h-4 w-4 text-red-500" />
      case "degraded":
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />
      default:
        return null
    }
  }

  return (
    <Card className={cn(
      "transition-all duration-200 hover:shadow-md",
      aiAnalysis && "border-l-4",
      aiAnalysis?.trend === "improving" && "border-l-green-500",
      aiAnalysis?.trend === "degrading" && "border-l-red-500",
      aiAnalysis?.trend === "stable" && "border-l-blue-500",
      className
    )}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-1">
        <CardTitle className="text-base font-semibold flex items-center gap-2">
          {title}
          {getStatusIcon()}
        </CardTitle>
        <div className="flex items-center gap-2">
          {aiAnalysis && showAIInsights && (
            <Badge variant="secondary" className="text-xs flex items-center gap-1">
              <Brain className="h-3 w-3" />
              AI
            </Badge>
          )}
          <div className={cn(
            "h-2 w-2 rounded-full",
            getStatusColor()
          )} />
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        <div className="flex items-center justify-between">
          <div className="text-lg font-semibold text-foreground">{value}</div>
          {aiAnalysis && getTrendIcon(aiAnalysis.trend)}
        </div>
        
        <p className="text-sm text-muted-foreground">{description}</p>
        
        {aiAnalysis && showAIInsights && (
          <div className="pt-2 border-t space-y-1">
            <div className="flex items-center justify-between text-xs">
              <span className={cn("font-medium", getTrendColor(aiAnalysis.trend))}>
                Trend: {aiAnalysis.trend}
              </span>
              <span className="text-muted-foreground">
                {Math.round(aiAnalysis.confidence * 100)}% confidence
              </span>
            </div>
            
            {aiAnalysis.prediction && (
              <p className="text-xs text-muted-foreground italic">
                AI Prediction: {aiAnalysis.prediction}
              </p>
            )}
            
            {aiAnalysis.recommendation && (
              <div className="bg-blue-50 dark:bg-blue-950/20 p-2 rounded text-xs">
                <span className="font-medium text-blue-700 dark:text-blue-300">
                  💡 Recommendation:
                </span>
                <p className="text-blue-600 dark:text-blue-400 mt-1">
                  {aiAnalysis.recommendation}
                </p>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// Mock AI analysis generator for demonstration
export function generateMockAIAnalysis(metricType: string): AIAnalysis {
  const analyses: Record<string, AIAnalysis> = {
    health: {
      trend: "stable",
      prediction: "Cluster health expected to remain stable for the next 24 hours",
      confidence: 0.89,
      recommendation: "Consider enabling auto-scaling for peak traffic periods"
    },
    score: {
      trend: "improving",
      prediction: "Health score likely to improve to 95-98% within next hour",
      confidence: 0.92,
      recommendation: "Continue current optimization strategies"
    },
    resources: {
      trend: "stable",
      prediction: "Resource usage patterns indicate stable performance",
      confidence: 0.87,
      recommendation: "Monitor memory usage during peak hours"
    },
    confidence: {
      trend: "improving",
      prediction: "AI analysis confidence increasing as more data is collected",
      confidence: 0.94
    }
  }

  return analyses[metricType] || {
    trend: "unknown",
    prediction: "Gathering data for analysis",
    confidence: 0.5
  }
}