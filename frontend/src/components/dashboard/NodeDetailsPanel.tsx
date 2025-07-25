import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"

interface NodeDetail {
  name: string
  ready: boolean
  cpu_allocatable: number
  cpu_percent: number
  memory_allocatable: number
  memory_percent: number
}

interface NodeMetric {
  name: string
  value: number
  unit?: string
  labels?: Record<string, string>
  timestamp?: string
  type?: string
}

interface NodeDetailsPanelProps {
  nodes: NodeDetail[]
  metrics?: NodeMetric[]
}

export function NodeDetailsPanel({ nodes }: NodeDetailsPanelProps) {
  const formatBytes = (bytes: number) => {
    const gb = bytes / (1024 * 1024 * 1024)
    return `${gb.toFixed(1)} GB`
  }

  const getHealthStatus = (ready: boolean, cpuPercent: number, memoryPercent: number) => {
    if (!ready) return { status: 'critical', color: 'bg-red-500', text: 'Not Ready' }
    if (cpuPercent > 80 || memoryPercent > 80) return { status: 'warning', color: 'bg-yellow-500', text: 'High Usage' }
    return { status: 'healthy', color: 'bg-green-500', text: 'Healthy' }
  }

  const getResourceColor = (percent: number) => {
    if (percent > 80) return "text-red-600"
    if (percent > 60) return "text-yellow-600"
    return "text-green-600"
  }

  if (nodes.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Node Details</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            No node information available
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
        {nodes.map((node, index) => {
          const healthStatus = getHealthStatus(node.ready, node.cpu_percent, node.memory_percent)
          const cpuAllocatable = node.cpu_allocatable / 1000 // Convert from millicores to cores
          const memoryAllocatable = formatBytes(node.memory_allocatable)
          
          return (
            <Card key={index} className="relative">
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-lg flex items-center gap-3">
                    <div className={`w-3 h-3 rounded-full ${healthStatus.color}`}></div>
                    {node.name}
                  </CardTitle>
                  <Badge variant={healthStatus.status === 'healthy' ? 'default' : 'destructive'}>
                    {healthStatus.text}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* CPU Usage */}
                <div>
                  <div className="flex justify-between items-center mb-2">
                    <span className="text-sm font-medium">CPU Usage</span>
                    <span className={`text-sm font-semibold ${getResourceColor(node.cpu_percent)}`}>
                      {Math.round(node.cpu_percent)}%
                    </span>
                  </div>
                  <Progress value={node.cpu_percent} className="h-2 mb-1" />
                  <div className="text-xs text-muted-foreground">
                    Allocatable: {cpuAllocatable} cores
                  </div>
                </div>

                <Separator />

                {/* Memory Usage */}
                <div>
                  <div className="flex justify-between items-center mb-2">
                    <span className="text-sm font-medium">Memory Usage</span>
                    <span className={`text-sm font-semibold ${getResourceColor(node.memory_percent)}`}>
                      {Math.round(node.memory_percent)}%
                    </span>
                  </div>
                  <Progress value={node.memory_percent} className="h-2 mb-1" />
                  <div className="text-xs text-muted-foreground">
                    Allocatable: {memoryAllocatable}
                  </div>
                </div>

                <Separator />

                {/* Node Status Details */}
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="text-muted-foreground">Status:</span>
                    <div className={`font-semibold ${node.ready ? 'text-green-600' : 'text-red-600'}`}>
                      {node.ready ? 'Ready' : 'Not Ready'}
                    </div>
                  </div>
                  <div>
                    <span className="text-muted-foreground">Role:</span>
                    <div className="font-semibold">
                      {node.name.includes('control-plane') ? 'Control Plane' : 'Worker'}
                    </div>
                  </div>
                </div>

                {/* Resource Efficiency Indicator */}
                <div className="bg-secondary/50 rounded-lg p-3 mt-4">
                  <div className="text-xs font-medium text-muted-foreground mb-2">
                    Resource Efficiency
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="flex-1">
                      <div className="text-sm font-semibold">
                        {node.cpu_percent < 20 && node.memory_percent < 20 ? 'Underutilized' :
                         node.cpu_percent > 80 || node.memory_percent > 80 ? 'High Usage' :
                         'Optimal'}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {node.cpu_percent < 20 && node.memory_percent < 20 ? 
                          'Consider workload consolidation' :
                         node.cpu_percent > 80 || node.memory_percent > 80 ? 
                          'Consider scaling or optimization' :
                          'Resource usage is balanced'}
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {/* Cluster-wide Node Summary */}
      <Card>
        <CardHeader>
          <CardTitle>Cluster Node Summary</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {nodes.filter(n => n.ready).length}
              </div>
              <div className="text-sm text-muted-foreground">Ready Nodes</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {nodes.reduce((sum, n) => sum + (n.cpu_allocatable / 1000), 0).toFixed(1)}
              </div>
              <div className="text-sm text-muted-foreground">Total CPU Cores</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {formatBytes(nodes.reduce((sum, n) => sum + n.memory_allocatable, 0))}
              </div>
              <div className="text-sm text-muted-foreground">Total Memory</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {Math.round(nodes.reduce((sum, n) => sum + n.cpu_percent, 0) / nodes.length)}%
              </div>
              <div className="text-sm text-muted-foreground">Avg CPU Usage</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}