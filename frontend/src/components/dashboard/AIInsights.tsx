import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"

export interface AIInsight {
  overall_health: string
  ai_confidence: number
  critical_issues: number
  trend_analysis?: string
  predicted_issues?: string[]
  top_recommendations?: Array<{
    title: string
    description: string
    impact?: string
    effort?: string
  }>
}

interface AIInsightsProps {
  insights: AIInsight | null
  loading?: boolean
  error?: string
}

export function AIInsights({ insights, loading, error }: AIInsightsProps) {
  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>🤖</span>
            AI Insights
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground animate-pulse">
            Loading AI insights...
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error || !insights) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>🤖</span>
            AI Insights
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Alert>
            <AlertTitle>AI Analysis Unavailable</AlertTitle>
            <AlertDescription>
              AI-powered insights are not available. Ensure Claude Code CLI is installed and accessible.
              <div className="mt-2">
                <code className="bg-secondary px-2 py-1 rounded text-sm">
                  npm install -g @anthropic-ai/claude-code
                </code>
              </div>
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    )
  }

  const getSeverityVariant = (criticalIssues: number): "default" | "destructive" => {
    if (criticalIssues > 0) return "destructive"
    return "default"
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <span>🤖</span>
          AI Insights
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <Alert variant={getSeverityVariant(insights.critical_issues)}>
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <AlertTitle>Cluster Health Assessment</AlertTitle>
              <AlertDescription className="mt-2">
                {insights.overall_health}
                {insights.trend_analysis && (
                  <p className="mt-2 italic">{insights.trend_analysis}</p>
                )}
              </AlertDescription>
            </div>
            <Badge variant="secondary" className="ml-4">
              AI Confidence: {Math.round(insights.ai_confidence * 100)}%
            </Badge>
          </div>
        </Alert>

        {insights.critical_issues > 0 && (
          <Alert variant="destructive">
            <AlertTitle>Critical Issues Detected</AlertTitle>
            <AlertDescription>
              Immediate attention required for {insights.critical_issues} critical{' '}
              {insights.critical_issues === 1 ? 'issue' : 'issues'}
            </AlertDescription>
          </Alert>
        )}

        {insights.predicted_issues && insights.predicted_issues.length > 0 && (
          <Alert>
            <AlertTitle>Predicted Issues</AlertTitle>
            <AlertDescription>
              <ul className="mt-2 space-y-1">
                {insights.predicted_issues.map((issue, index) => (
                  <li key={index} className="flex items-start">
                    <span className="mr-2">•</span>
                    {issue}
                  </li>
                ))}
              </ul>
            </AlertDescription>
          </Alert>
        )}

        {insights.top_recommendations && insights.top_recommendations.length > 0 && (
          <div>
            <h4 className="text-base font-semibold mb-3 flex items-center gap-2">
              <span>🎯</span>
              Top Recommendations
            </h4>
            <div className="space-y-3">
              {insights.top_recommendations.slice(0, 3).map((rec, index) => (
                <div
                  key={index}
                  className="bg-secondary/50 rounded-lg p-4 border-l-4 border-l-green-500"
                >
                  <h5 className="font-semibold text-sm text-green-600 mb-1">
                    {rec.title}
                  </h5>
                  <p className="text-sm text-muted-foreground">{rec.description}</p>
                  {(rec.impact || rec.effort) && (
                    <div className="mt-2 text-xs text-muted-foreground">
                      {rec.impact && <span>Impact: {rec.impact}</span>}
                      {rec.impact && rec.effort && <span> • </span>}
                      {rec.effort && <span>Effort: {rec.effort}</span>}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}