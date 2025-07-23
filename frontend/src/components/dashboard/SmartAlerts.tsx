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

export function SmartAlerts() {
  const [alerts, setAlerts] = useState<SmartAlert[]>([])
  const [insights, setInsights] = useState<AlertInsights | null>(null)
  const [loading, setLoading] = useState(true)
  // const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchAlerts = async () => {
      try {
        setLoading(true)
        const response = await fetch('/api/v1/ai/alerts/insights')
        
        if (!response.ok) {
          throw new Error('Failed to fetch smart alerts')
        }
        
        const data = await response.json()
        setAlerts(data.alerts || [])
        setInsights(data.insights || null)
        // setError(null)
      } catch (err) {
        console.error('Failed to load smart alerts:', err)
        // setError(err instanceof Error ? err.message : 'Unknown error')
        // Generate mock data for demonstration
        setAlerts([
          {
            id: 'alert-1',
            type: 'threshold',
            severity: 'medium',
            title: 'Memory Usage Above Normal',
            description: 'Node desktop-worker memory usage has exceeded 75% for the past 15 minutes',
            resource: 'desktop-worker',
            timestamp: new Date().toISOString(),
            correlation_score: 0.85,
            noise_reduced: true,
            similar_alerts: 3,
            suggested_actions: [
              'Check for memory leaks in running pods',
              'Consider scaling the node pool',
              'Review memory limits on high-usage pods'
            ]
          },
          {
            id: 'alert-2',
            type: 'anomaly',
            severity: 'low',
            title: 'Unusual Network Traffic Pattern',
            description: 'Detected abnormal traffic patterns in service communication',
            resource: 'cluster-network',
            timestamp: new Date(Date.now() - 300000).toISOString(),
            correlation_score: 0.72,
            noise_reduced: false,
            similar_alerts: 1,
            suggested_actions: [
              'Review network policies',
              'Check for new deployments affecting traffic',
              'Monitor for potential security issues'
            ]
          }
        ])
        setInsights({
          total_alerts: 2,
          alerts_by_severity: { low: 1, medium: 1, high: 0, critical: 0 },
          noise_reduction_rate: 0.65,
          correlation_success_rate: 0.78,
          top_alert_sources: [
            { source: 'Node Monitoring', count: 5, trend: 'stable' },
            { source: 'Pod Health', count: 3, trend: 'decreasing' },
            { source: 'Network Monitoring', count: 2, trend: 'increasing' }
          ],
          smart_grouping: [
            {
              group_name: 'Resource Pressure',
              alert_count: 4,
              common_cause: 'Increased workload demands',
              recommendation: 'Consider horizontal scaling'
            }
          ]
        })
      } finally {
        setLoading(false)
      }
    }

    fetchAlerts()
    const interval = setInterval(fetchAlerts, 30000) // Refresh every 30 seconds
    
    return () => clearInterval(interval)
  }, [])

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'border-red-500 bg-red-50'
      case 'high': return 'border-orange-500 bg-orange-50'
      case 'medium': return 'border-yellow-500 bg-yellow-50'
      case 'low': return 'border-blue-500 bg-blue-50'
      default: return 'border-gray-500 bg-gray-50'
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

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'anomaly': return '🔍'
      case 'threshold': return '📊'
      case 'pattern': return '🔄'
      case 'prediction': return '🔮'
      default: return '⚠️'
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
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>🚨</span>
            Smart Alerts
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground animate-pulse">
            Loading intelligent alerts...
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Alert Insights Summary */}
      {insights && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <span>📊</span>
              Alert Intelligence Summary
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
              <div className="text-center">
                <div className="text-2xl font-bold text-primary">
                  {insights.total_alerts}
                </div>
                <div className="text-sm text-muted-foreground">Active Alerts</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-green-600">
                  {Math.round(insights.noise_reduction_rate * 100)}%
                </div>
                <div className="text-sm text-muted-foreground">Noise Reduced</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-blue-600">
                  {Math.round(insights.correlation_success_rate * 100)}%
                </div>
                <div className="text-sm text-muted-foreground">Correlation Rate</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-purple-600">
                  {insights.smart_grouping.length}
                </div>
                <div className="text-sm text-muted-foreground">Alert Groups</div>
              </div>
            </div>

            {/* Top Alert Sources */}
            <div className="space-y-2">
              <h4 className="font-semibold text-sm">Top Alert Sources</h4>
              {insights.top_alert_sources.map((source, index) => (
                <div key={index} className="flex items-center justify-between bg-secondary/50 rounded p-2">
                  <span className="text-sm">{source.source}</span>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">
                      {source.count}
                    </Badge>
                    <span className="text-sm">
                      {getTrendIcon(source.trend)}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Active Smart Alerts */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>🚨</span>
            Active Smart Alerts
          </CardTitle>
        </CardHeader>
        <CardContent>
          {alerts.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              No active alerts - your cluster is running smoothly! ✨
            </div>
          ) : (
            <div className="space-y-4">
              {alerts.map((alert) => (
                <Alert key={alert.id} className={getSeverityColor(alert.severity)}>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <AlertTitle className="flex items-center gap-2 mb-2">
                        {getTypeIcon(alert.type)}
                        {alert.title}
                        <Badge variant={getSeverityVariant(alert.severity)} className="text-xs">
                          {alert.severity}
                        </Badge>
                        {alert.noise_reduced && (
                          <Badge variant="outline" className="text-xs">
                            Noise Reduced
                          </Badge>
                        )}
                      </AlertTitle>
                      <AlertDescription className="space-y-3">
                        <p>{alert.description}</p>
                        
                        <div className="flex items-center gap-4 text-sm">
                          <span><strong>Resource:</strong> {alert.resource}</span>
                          <span><strong>Correlation:</strong> {Math.round(alert.correlation_score * 100)}%</span>
                          {alert.similar_alerts > 0 && (
                            <span><strong>Similar:</strong> {alert.similar_alerts} alerts</span>
                          )}
                        </div>

                        {alert.suggested_actions.length > 0 && (
                          <div>
                            <h5 className="font-semibold text-sm mb-2">Suggested Actions:</h5>
                            <ul className="space-y-1">
                              {alert.suggested_actions.map((action, index) => (
                                <li key={index} className="flex items-start gap-2 text-sm">
                                  <span className="mt-1">•</span>
                                  <span>{action}</span>
                                </li>
                              ))}
                            </ul>
                          </div>
                        )}
                      </AlertDescription>
                    </div>
                    <div className="ml-4 text-right">
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

      {/* Smart Grouping */}
      {insights?.smart_grouping && insights.smart_grouping.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <span>🔗</span>
              Smart Alert Grouping
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {insights.smart_grouping.map((group, index) => (
                <div key={index} className="border rounded-lg p-4">
                  <div className="flex items-center justify-between mb-2">
                    <h4 className="font-semibold">{group.group_name}</h4>
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