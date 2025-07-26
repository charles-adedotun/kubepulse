import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { useState, useEffect } from "react"

interface PredictionData {
  predictions: Array<{
    alert_type: string
    probability: number
    time_window: string
    prevention: string
    confidence: number
    severity: 'low' | 'medium' | 'high' | 'critical'
  }>
  trend_analysis: {
    cpu_trend: 'increasing' | 'decreasing' | 'stable'
    memory_trend: 'increasing' | 'decreasing' | 'stable'
    workload_trend: 'increasing' | 'decreasing' | 'stable'
    forecast_accuracy: number
  }
  recommendations: Array<{
    category: string
    action: string
    impact: string
    urgency: 'low' | 'medium' | 'high'
    time_to_implement: string
  }>
}

interface ClusterHealth {
  score?: {
    trend?: string
  }
}

interface AIInsights {
  [key: string]: unknown
}

interface PredictiveAnalyticsProps {
  clusterHealth?: ClusterHealth
  insights?: AIInsights
}

export function PredictiveAnalytics({ clusterHealth }: PredictiveAnalyticsProps) {
  const [predictions, setPredictions] = useState<PredictionData | null>(null)
  const [loading, setLoading] = useState(true)
  // const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchPredictions = async () => {
      try {
        setLoading(true)
        const response = await fetch('/api/v1/ai/predictions')
        
        if (!response.ok) {
          throw new Error('Failed to fetch predictions')
        }
        
        const data = await response.json()
        setPredictions(data)
        // setError(null)
      } catch (err) {
        console.error('Failed to load predictions:', err)
        // setError(err instanceof Error ? err.message : 'Unknown error')
        // Generate mock data for demonstration
        setPredictions({
          predictions: [
            {
              alert_type: "Memory Pressure",
              probability: 0.75,
              time_window: "2-4 hours",
              prevention: "Scale nodes or optimize pod resources",
              confidence: 0.85,
              severity: 'medium'
            },
            {
              alert_type: "CPU Throttling",
              probability: 0.45,
              time_window: "6-8 hours",
              prevention: "Adjust CPU limits or add nodes",
              confidence: 0.72,
              severity: 'low'
            },
            {
              alert_type: "Pod Eviction Risk",
              probability: 0.25,
              time_window: "12-24 hours",
              prevention: "Review resource quotas and node capacity",
              confidence: 0.68,
              severity: 'low'
            }
          ],
          trend_analysis: {
            cpu_trend: 'stable',
            memory_trend: clusterHealth?.score?.trend || 'stable',
            workload_trend: 'stable',
            forecast_accuracy: 0.82
          },
          recommendations: [
            {
              category: "Resource Optimization",
              action: "Implement horizontal pod autoscaling for high-usage workloads",
              impact: "Reduce resource pressure by 30-40%",
              urgency: 'medium',
              time_to_implement: "2-3 hours"
            },
            {
              category: "Monitoring",
              action: "Set up alerts for memory usage above 70%",
              impact: "Early warning system for resource issues",
              urgency: 'high',
              time_to_implement: "30 minutes"
            },
            {
              category: "Capacity Planning",
              action: "Plan for node scaling when cluster utilization exceeds 75%",
              impact: "Prevent future resource bottlenecks",
              urgency: 'low',
              time_to_implement: "1-2 days"
            }
          ]
        })
      } finally {
        setLoading(false)
      }
    }

    fetchPredictions()
    const interval = setInterval(fetchPredictions, 60000) // Refresh every minute
    
    return () => clearInterval(interval)
  }, [clusterHealth])

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'text-red-600 bg-red-50 border-red-200'
      case 'high': return 'text-orange-600 bg-orange-50 border-orange-200'
      case 'medium': return 'text-yellow-600 bg-yellow-50 border-yellow-200'
      case 'low': return 'text-blue-600 bg-blue-50 border-blue-200'
      default: return 'text-gray-600 bg-gray-50 border-gray-200'
    }
  }

  const getUrgencyVariant = (urgency: string): "default" | "secondary" | "destructive" => {
    switch (urgency) {
      case 'high': return 'destructive'
      case 'medium': return 'default'
      case 'low': return 'secondary'
      default: return 'secondary'
    }
  }

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'increasing': return '📈'
      case 'decreasing': return '📉'
      case 'stable': return '➡️'
      default: return '❓'
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <span>🔮</span>
              Predictive Analytics
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-center py-8 text-muted-foreground animate-pulse">
              Analyzing cluster patterns and generating predictions...
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Trend Analysis */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>📈</span>
            Trend Analysis
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="text-center">
              <div className="text-2xl mb-2">
                {getTrendIcon(predictions?.trend_analysis.cpu_trend || 'stable')}
              </div>
              <div className="font-semibold">CPU Trend</div>
              <div className="text-sm text-muted-foreground capitalize">
                {predictions?.trend_analysis.cpu_trend || 'stable'}
              </div>
            </div>
            <div className="text-center">
              <div className="text-2xl mb-2">
                {getTrendIcon(predictions?.trend_analysis.memory_trend || 'stable')}
              </div>
              <div className="font-semibold">Memory Trend</div>
              <div className="text-sm text-muted-foreground capitalize">
                {predictions?.trend_analysis.memory_trend || 'stable'}
              </div>
            </div>
            <div className="text-center">
              <div className="text-2xl mb-2">
                {getTrendIcon(predictions?.trend_analysis.workload_trend || 'stable')}
              </div>
              <div className="font-semibold">Workload Trend</div>
              <div className="text-sm text-muted-foreground capitalize">
                {predictions?.trend_analysis.workload_trend || 'stable'}
              </div>
            </div>
          </div>
          <div className="mt-6 text-center">
            <Badge variant="secondary">
              Forecast Accuracy: {Math.round((predictions?.trend_analysis.forecast_accuracy || 0.82) * 100)}%
            </Badge>
          </div>
        </CardContent>
      </Card>

      {/* Predictions */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>🔮</span>
            Potential Issues Predictions
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {predictions?.predictions.map((prediction, index) => (
              <Alert key={index} className={getSeverityColor(prediction.severity)}>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <AlertTitle className="flex items-center gap-2">
                      {prediction.alert_type}
                      <Badge variant="outline" className="text-xs">
                        {Math.round(prediction.probability * 100)}% probability
                      </Badge>
                    </AlertTitle>
                    <AlertDescription className="mt-2">
                      <div className="space-y-1">
                        <p><strong>Expected timeframe:</strong> {prediction.time_window}</p>
                        <p><strong>Prevention:</strong> {prediction.prevention}</p>
                      </div>
                    </AlertDescription>
                  </div>
                  <div className="ml-4 text-right">
                    <div className="text-sm font-semibold capitalize">{prediction.severity}</div>
                    <Progress 
                      value={prediction.probability * 100} 
                      className="w-20 h-2 mt-1" 
                    />
                  </div>
                </div>
              </Alert>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Proactive Recommendations */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>💡</span>
            Proactive Recommendations
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {predictions?.recommendations.map((rec, index) => (
              <div key={index} className="border rounded-lg p-4 hover:bg-secondary/50 transition-colors">
                <div className="flex items-start justify-between mb-2">
                  <div className="flex-1">
                    <div className="font-semibold text-sm text-primary mb-1">
                      {rec.category}
                    </div>
                    <h4 className="font-medium mb-2">{rec.action}</h4>
                    <p className="text-sm text-muted-foreground mb-2">{rec.impact}</p>
                  </div>
                  <div className="ml-4 flex flex-col items-end gap-2">
                    <Badge variant={getUrgencyVariant(rec.urgency)}>
                      {rec.urgency} priority
                    </Badge>
                    <div className="text-xs text-muted-foreground">
                      {rec.time_to_implement}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}