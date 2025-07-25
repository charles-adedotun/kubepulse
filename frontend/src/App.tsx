import { Header } from '@/components/layout/Header'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { StatusCard } from '@/components/dashboard/StatusCard'
import { HealthChecksTable } from '@/components/dashboard/HealthChecksTable'
import { EnhancedMetricsGrid, type EnhancedMetric } from '@/components/dashboard/EnhancedMetricsGrid'
import { AIInsights } from '@/components/dashboard/AIInsights'
import { NodeDetailsPanel } from '@/components/dashboard/NodeDetailsPanel'
import { PredictiveAnalytics } from '@/components/dashboard/PredictiveAnalytics'
import { SmartAlerts } from '@/components/dashboard/SmartAlerts'
import { useWebSocket } from '@/hooks/useWebSocket'
import { useAIInsights } from '@/hooks/useAIInsights'
import { useSystemTheme } from '@/hooks/useSystemTheme'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useState } from 'react'

interface KubernetesContext {
  name: string
  cluster_name: string
  namespace: string
}

interface MetricLabels {
  [key: string]: string
  node?: string
}


interface NodeDetail {
  ready: boolean
  cpu_percent?: number
  memory_percent?: number
  [key: string]: unknown
}


interface ScoreData {
  weighted: number
  trend?: string
  forecast?: string
}
import { config } from '@/config'
import { useRuntimeConfig } from '@/hooks/useRuntimeConfig'

function App() {
  // Load runtime configuration from server
  useRuntimeConfig()
  
  const [currentContext, setCurrentContext] = useState<KubernetesContext | null>(null)
  const { data, connectionStatus } = useWebSocket()
  const { insights, loading: aiLoading, error: aiError } = useAIInsights()
  const [activeTab, setActiveTab] = useState('overview')
  useSystemTheme() // This hook handles applying dark class to document

  const handleContextChange = (context: KubernetesContext) => {
    setCurrentContext(context)
    // The websocket connection will automatically receive updates
    // for the new context from the server
  }

  // Extract metrics from all checks with enhanced metadata
  const allMetrics = data?.checks?.flatMap(check => 
    check.metrics?.map(metric => ({
      name: metric.name,
      value: metric.value,
      unit: metric.unit,
      labels: metric.labels,
      timestamp: metric.timestamp,
      type: metric.type,
      checkName: check.name
    })) || []
  ) || []

  // Extract node details for enhanced visualization
  const nodeDetails: NodeDetail[] = data?.checks?.find(check => check.name === 'node-health')?.details?.nodes || []
  
  // Calculate cluster summary stats
  const clusterStats = {
    totalNodes: nodeDetails.length,
    healthyNodes: nodeDetails.filter(node => node.ready).length,
    avgCpuUsage: nodeDetails.reduce((sum, node) => sum + (node.cpu_percent || 0), 0) / (nodeDetails.length || 1),
    avgMemoryUsage: nodeDetails.reduce((sum, node) => sum + (node.memory_percent || 0), 0) / (nodeDetails.length || 1)
  }

  return (
    <div className="min-h-screen bg-background">
      <Header connectionStatus={connectionStatus} onContextChange={handleContextChange} />
      <DashboardLayout>
        {/* Context Info Bar */}
        {currentContext && (
          <div className="mb-6 p-4 bg-muted/50 rounded-lg border">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">Current Context:</span>
              <span className="text-sm font-semibold">{currentContext.name}</span>
              <span className="text-sm text-muted-foreground">•</span>
              <span className="text-sm text-muted-foreground">{currentContext.cluster_name}</span>
              <span className="text-sm text-muted-foreground">•</span>
              <span className="text-sm text-muted-foreground">Namespace: {currentContext.namespace}</span>
            </div>
          </div>
        )}

        {/* Enhanced Status Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <StatusCard
            title="Overall Health"
            value={data?.status?.toUpperCase() || 'UNKNOWN'}
            description={data ? `Last updated: ${new Date(data.timestamp).toLocaleTimeString()}` : 'Loading...'}
            status={data?.status || 'unknown'}
          />
          <StatusCard
            title="Health Score"
            value={data?.score ? `${Math.round(data.score.weighted)}%` : '--'}
            description={`Trend: ${(data?.score as ScoreData)?.trend || 'Unknown'} | Forecast: ${(data?.score as ScoreData)?.forecast || 'Unknown'}`}
            status="healthy"
          />
          <StatusCard
            title="Cluster Resources"
            value={`${clusterStats.healthyNodes}/${clusterStats.totalNodes}`}
            description={`Nodes Ready | Avg CPU: ${Math.round(clusterStats.avgCpuUsage)}%`}
            status="healthy"
          />
          {config.features.aiInsights && (
            <StatusCard
              title="AI Confidence"
              value={insights ? `${Math.round((insights.ai_confidence || 0) * 100)}%` : '--'}
              description={`Critical Issues: ${insights?.critical_issues || 0}`}
              status={(insights?.critical_issues || 0) > 0 ? 'unhealthy' : 'healthy'}
            />
          )}
        </div>

        {/* Main Dashboard Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
          <TabsList className={`grid w-full grid-cols-${1 + (config.features.nodeDetails ? 1 : 0) + (config.features.aiInsights ? 1 : 0) + (config.features.predictiveAnalytics ? 1 : 0)}`}>
            <TabsTrigger value="overview" data-state={activeTab === 'overview' ? 'active' : ''}>Overview</TabsTrigger>
            {config.features.nodeDetails && (
              <TabsTrigger value="nodes" data-state={activeTab === 'nodes' ? 'active' : ''}>Node Details</TabsTrigger>
            )}
            {config.features.aiInsights && (
              <TabsTrigger value="ai-insights" data-state={activeTab === 'ai-insights' ? 'active' : ''}>AI Insights</TabsTrigger>
            )}
            {config.features.predictiveAnalytics && (
              <TabsTrigger value="predictions" data-state={activeTab === 'predictions' ? 'active' : ''}>Predictions</TabsTrigger>
            )}
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            {/* Health Checks */}
            <HealthChecksTable 
              checks={data?.checks?.map(check => ({
                name: check.name,
                status: check.status,
                message: check.message,
                timestamp: check.timestamp,
                duration: check.duration
              })) || []}
            />

            {/* Enhanced Metrics */}
            <EnhancedMetricsGrid 
              metrics={allMetrics}
              clusterStats={clusterStats}
            />
          </TabsContent>

          {config.features.nodeDetails && (
            <TabsContent value="nodes" className="space-y-6">
              <NodeDetailsPanel 
                nodes={nodeDetails}
                metrics={allMetrics.filter(m => m.labels?.node)}
              />
            </TabsContent>
          )}

          {config.features.aiInsights && (
            <TabsContent value="ai-insights" className="space-y-6">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <AIInsights 
                  insights={insights}
                  loading={aiLoading}
                  error={aiError || undefined}
                />
                {config.features.smartAlerts && <SmartAlerts />}
              </div>
            </TabsContent>
          )}

          {config.features.predictiveAnalytics && (
            <TabsContent value="predictions" className="space-y-6">
              <PredictiveAnalytics 
                clusterHealth={data}
                insights={insights}
              />
            </TabsContent>
          )}
        </Tabs>
      </DashboardLayout>
    </div>
  )
}

export default App
