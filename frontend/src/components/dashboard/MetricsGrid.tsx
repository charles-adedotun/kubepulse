import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export interface Metric {
  name: string
  value: number
  unit?: string
}

interface MetricsGridProps {
  metrics: Metric[]
}

export function MetricsGrid({ metrics }: MetricsGridProps) {
  if (metrics.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>📊</span>
            Key Metrics
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            No metrics available
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <span>📊</span>
          Key Metrics
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {metrics.slice(0, 8).map((metric, index) => (
            <div
              key={index}
              className="bg-secondary/50 rounded-lg p-4 text-center transition-colors hover:bg-secondary/70"
            >
              <div className="text-xl font-semibold text-primary">
                {metric.value}
                {metric.unit && <span className="text-sm ml-1">{metric.unit}</span>}
              </div>
              <div className="text-sm text-muted-foreground mt-1 uppercase">
                {metric.name.replace(/_/g, ' ')}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}