import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Badge } from "@/components/ui/badge"

export interface EnhancedMetric {
  name: string
  value: number
  unit?: string
  labels?: Record<string, string>
  timestamp?: string
  type?: string
  checkName?: string
}

interface ClusterStats {
  totalNodes: number
  healthyNodes: number
  avgCpuUsage: number
  avgMemoryUsage: number
}

interface EnhancedMetricsGridProps {
  metrics: EnhancedMetric[]
  clusterStats: ClusterStats
}

export function EnhancedMetricsGrid({ metrics, clusterStats }: EnhancedMetricsGridProps) {
  // Group metrics by type for better organization
  const cpuMetrics = metrics.filter(m => m.name.includes('cpu'))
  const memoryMetrics = metrics.filter(m => m.name.includes('memory'))
  const nodeMetrics = metrics.filter(m => m.name.includes('node'))
  const podMetrics = metrics.filter(m => m.name.includes('pod'))
  const serviceMetrics = metrics.filter(m => m.name.includes('service'))

  const getStatusColor = (value: number, threshold = 80) => {
    if (value > threshold) return "text-red-600"
    if (value > 60) return "text-yellow-600"
    return "text-green-600"
  }

  const formatValue = (value: number, unit?: string) => {
    if (unit === '%' || (typeof value === 'number' && value > 0 && value < 1)) {
      return `${Math.round(value * 100)}%`
    }
    if (value > 1000000) {
      return `${(value / 1000000).toFixed(1)}M`
    }
    if (value > 1000) {
      return `${(value / 1000).toFixed(1)}K`
    }
    return Math.round(value).toString()
  }

  return (
    <div className="space-y-6">
      {/* Cluster Overview */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <span>🎯</span>
            Cluster Overview
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {clusterStats.healthyNodes}/{clusterStats.totalNodes}
              </div>
              <div className="text-sm text-muted-foreground">Healthy Nodes</div>
            </div>
            <div className="text-center">
              <div className={`text-2xl font-bold ${getStatusColor(clusterStats.avgCpuUsage)}`}>
                {Math.round(clusterStats.avgCpuUsage)}%
              </div>
              <div className="text-sm text-muted-foreground">Avg CPU Usage</div>
              <Progress value={clusterStats.avgCpuUsage} className="mt-2 h-2" />
            </div>
            <div className="text-center">
              <div className={`text-2xl font-bold ${getStatusColor(clusterStats.avgMemoryUsage)}`}>
                {Math.round(clusterStats.avgMemoryUsage)}%
              </div>
              <div className="text-sm text-muted-foreground">Avg Memory Usage</div>
              <Progress value={clusterStats.avgMemoryUsage} className="mt-2 h-2" />
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {podMetrics.find(m => m.name === 'pod_running')?.value || 0}
              </div>
              <div className="text-sm text-muted-foreground">Running Pods</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Resource Utilization by Node */}
      {cpuMetrics.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <span>💻</span>
              Resource Utilization by Node
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {Array.from(new Set(cpuMetrics.map(m => m.labels?.node).filter(Boolean))).map(nodeName => {
                const cpuMetric = cpuMetrics.find(m => m.labels?.node === nodeName)
                const memoryMetric = memoryMetrics.find(m => m.labels?.node === nodeName)
                
                return (
                  <div key={nodeName} className="bg-secondary/50 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <h4 className="font-semibold flex items-center gap-2">
                        <span className="w-2 h-2 bg-green-500 rounded-full"></span>
                        {nodeName}
                      </h4>
                      <Badge variant="outline" className="text-xs">
                        {new Date(cpuMetric?.timestamp || '').toLocaleTimeString()}
                      </Badge>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="flex justify-between items-center mb-2">
                          <span className="text-sm text-muted-foreground">CPU Usage</span>
                          <span className={`font-semibold ${getStatusColor(cpuMetric?.value || 0)}`}>
                            {formatValue(cpuMetric?.value || 0, '%')}
                          </span>
                        </div>
                        <Progress value={cpuMetric?.value || 0} className="h-2" />
                      </div>
                      <div>
                        <div className="flex justify-between items-center mb-2">
                          <span className="text-sm text-muted-foreground">Memory Usage</span>
                          <span className={`font-semibold ${getStatusColor(memoryMetric?.value || 0)}`}>
                            {formatValue(memoryMetric?.value || 0, '%')}
                          </span>
                        </div>
                        <Progress value={memoryMetric?.value || 0} className="h-2" />
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Workload Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Nodes */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <span>🖥️</span>
              Nodes
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {nodeMetrics.map((metric, index) => (
                <div key={index} className="flex justify-between items-center">
                  <span className="text-sm text-muted-foreground">
                    {metric.name.replace(/_/g, ' ').replace(/node/g, '').trim()}
                  </span>
                  <span className="font-semibold">
                    {formatValue(metric.value, metric.unit)}
                  </span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Pods */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <span>📦</span>
              Pods
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {podMetrics.map((metric, index) => (
                <div key={index} className="flex justify-between items-center">
                  <span className="text-sm text-muted-foreground">
                    {metric.name.replace(/_/g, ' ').replace(/pod/g, '').trim()}
                  </span>
                  <span className="font-semibold">
                    {formatValue(metric.value, metric.unit)}
                  </span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Services */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <span>🔗</span>
              Services
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {serviceMetrics.map((metric, index) => (
                <div key={index} className="flex justify-between items-center">
                  <span className="text-sm text-muted-foreground">
                    {metric.name.replace(/_/g, ' ').replace(/service/g, '').trim()}
                  </span>
                  <span className="font-semibold">
                    {formatValue(metric.value, metric.unit)}
                  </span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}