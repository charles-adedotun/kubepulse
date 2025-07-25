import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useState, useEffect } from "react"

interface SmartAlert {
  id: string
  type: 'anomaly' | 'threshold' | 'pattern' | 'prediction'
  severity: 'low' | 'medium' | 'high' | 'critical'
  title: string
  description: string
  resource: string
  timestamp: string
  correlation_score: number
  noise_reduced: boolean
  similar_alerts: number
  suggested_actions: string[]
}

interface AlertInsights {
  total_alerts: number
  alerts_by_severity: Record<string, number>
  noise_reduction_rate: number
  correlation_success_rate: number
  top_alert_sources: Array<{
    source: string
    count: number
    trend: 'increasing' | 'decreasing' | 'stable'
  }>
  smart_grouping: Array<{
    group_name: string
    alert_count: number
    common_cause: string
    recommendation: string
  }>
}

// Type guard to validate smart alerts response
function isValidSmartAlertsResponse(data: any): data is { alerts: SmartAlert[], insights: AlertInsights } {
  if (!data || typeof data !== 'object') {
    return false
  }

  // Check alerts array
  if (!Array.isArray(data.alerts)) {
    return false
  }

  // Validate each alert structure
  for (const alert of data.alerts) {
    if (!alert || typeof alert !== 'object' ||
        typeof alert.id !== 'string' ||
        typeof alert.title !== 'string' ||
        typeof alert.description !== 'string' ||
        typeof alert.severity !== 'string' ||
        !Array.isArray(alert.suggested_actions)) {
      return false
    }
  }

  // Check insights structure
  if (!data.insights || typeof data.insights !== 'object' ||
      typeof data.insights.total_alerts !== 'number' ||
      !data.insights.alerts_by_severity ||
      typeof data.insights.noise_reduction_rate !== 'number') {
    return false
  }

  return true
}

export function SmartAlerts() {
  const [alerts, setAlerts] = useState<SmartAlert[]>([])
  const [insights, setInsights] = useState<AlertInsights | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchAlerts = async () => {
      try {
        setLoading(true)
        const response = await fetch('/api/v1/ai/alerts/insights')
        
        if (!response.ok) {
          throw new Error('Failed to fetch smart alerts')
        }
        
        const data = await response.json()
        
        // Validate response structure before using it
        if (isValidSmartAlertsResponse(data)) {
          setAlerts(data.alerts)
          setInsights(data.insights)
        } else {
          console.warn('Received invalid smart alerts data structure:', data)
          setAlerts([])
          setInsights(null)
        }
      } catch (err) {
        console.error('Failed to load smart alerts:', err)
        setAlerts([])
        setInsights(null)
      } finally {
        setLoading(false)
      }
    }

    fetchAlerts()
    const interval = setInterval(fetchAlerts, 30000)
    
    return () => clearInterval(interval)
  }, [])

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'border-destructive'
      case 'high': return 'border-destructive/80'
      case 'medium': return 'border-yellow-500'
      case 'low': return 'border-muted'
      default: return 'border-muted'
    }
  }

  const getSeverityVariant = (severity: string): "default" | "secondary" | "destructive" => {
    switch (severity) {
      case 'critical':
      case 'high':
        return 'destructive'
      case 'medium':
        return 'default'
      default:
        return 'secondary'
    }
  }

  const getTrendIndicator = (trend: string) => {
    switch (trend) {
      case 'increasing': return '↑'
      case 'decreasing': return '↓'
      case 'stable': return '→'
      default: return '-'
    }
  }

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Smart Alerts</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            Loading alerts...
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Alert Summary */}
      {insights && (
        <Card>
          <CardHeader>
            <CardTitle>Alert Summary</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
              <div className="text-center">
                <div className="text-xl font-semibold">
                  {insights.total_alerts}
                </div>
                <div className="text-sm text-muted-foreground">Active Alerts</div>
              </div>
              <div className="text-center">
                <div className="text-xl font-semibold">
                  {Math.round(insights.noise_reduction_rate * 100)}%
                </div>
                <div className="text-sm text-muted-foreground">Noise Reduced</div>
              </div>
              <div className="text-center">
                <div className="text-xl font-semibold">
                  {Math.round(insights.correlation_success_rate * 100)}%
                </div>
                <div className="text-sm text-muted-foreground">Correlation Rate</div>
              </div>
              <div className="text-center">
                <div className="text-xl font-semibold">
                  {insights.smart_grouping.length}
                </div>
                <div className="text-sm text-muted-foreground">Alert Groups</div>
              </div>
            </div>

            {/* Top Alert Sources */}
            <div className="space-y-2">
              <h4 className="text-base font-semibold">Top Alert Sources</h4>
              {insights.top_alert_sources.map((source, index) => (
                <div key={index} className="flex items-center justify-between bg-secondary/50 rounded p-2">
                  <span className="text-sm">{source.source}</span>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">
                      {source.count}
                    </Badge>
                    <span className="text-sm font-semibold">
                      {getTrendIndicator(source.trend)}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Active Alerts */}
      <Card>
        <CardHeader>
          <CardTitle>Active Alerts</CardTitle>
        </CardHeader>
        <CardContent>
          {alerts.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              No active alerts
            </div>
          ) : (
            <div className="space-y-4">
              {alerts.map((alert) => (
                <Alert key={alert.id} className={getSeverityColor(alert.severity)}>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <AlertTitle className="mb-2">
                        {alert.title}
                        <Badge variant={getSeverityVariant(alert.severity)} className="ml-2">
                          {alert.severity}
                        </Badge>
                      </AlertTitle>
                      <AlertDescription className="space-y-2">
                        <p className="text-sm">{alert.description}</p>
                        
                        <div className="flex items-center gap-4 text-sm">
                          <span><strong>Resource:</strong> {alert.resource}</span>
                          <span><strong>Correlation:</strong> {Math.round(alert.correlation_score * 100)}%</span>
                          {alert.similar_alerts > 0 && (
                            <span><strong>Similar:</strong> {alert.similar_alerts} alerts</span>
                          )}
                        </div>

                        {alert.suggested_actions.length > 0 && (
                          <div className="mt-4">
                            <h5 className="font-semibold text-sm mb-2">Suggested Actions:</h5>
                            <ul className="space-y-1">
                              {alert.suggested_actions.map((action, index) => (
                                <li key={index} className="flex items-start gap-2 text-sm">
                                  <span className="mt-0.5">•</span>
                                  <span>{action}</span>
                                </li>
                              ))}
                            </ul>
                          </div>
                        )}
                      </AlertDescription>
                    </div>
                    <div className="ml-4">
                      <div className="text-xs text-muted-foreground mb-2">
                        {new Date(alert.timestamp).toLocaleTimeString()}
                      </div>
                      <Button size="sm" variant="outline">
                        Investigate
                      </Button>
                    </div>
                  </div>
                </Alert>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Alert Groups */}
      {insights?.smart_grouping && insights.smart_grouping.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Alert Groups</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {insights.smart_grouping.map((group, index) => (
                <div key={index} className="border rounded-lg p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h4 className="text-base font-semibold">{group.group_name}</h4>
                    <Badge variant="secondary">
                      {group.alert_count} alerts
                    </Badge>
                  </div>
                  <p className="text-sm text-muted-foreground mb-2">
                    <strong>Common Cause:</strong> {group.common_cause}
                  </p>
                  <p className="text-sm">
                    <strong>Recommendation:</strong> {group.recommendation}
                  </p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}