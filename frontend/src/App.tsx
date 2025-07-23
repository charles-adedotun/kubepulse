import { Header } from '@/components/layout/Header'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { StatusCard } from '@/components/dashboard/StatusCard'
import { HealthChecksTable } from '@/components/dashboard/HealthChecksTable'
import { EnhancedMetricsGrid } from '@/components/dashboard/EnhancedMetricsGrid'
import { AIInsights } from '@/components/dashboard/AIInsights'
import { NodeDetailsPanel } from '@/components/dashboard/NodeDetailsPanel'
import { PredictiveAnalytics } from '@/components/dashboard/PredictiveAnalytics'
import { SmartAlerts } from '@/components/dashboard/SmartAlerts'
import { useWebSocket } from '@/hooks/useWebSocket'
import { useAIInsights } from '@/hooks/useAIInsights'
import { useSystemTheme } from '@/hooks/useSystemTheme'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useState } from 'react'

function App() {
  const { data, connectionStatus } = useWebSocket()
  const { insights, loading: aiLoading, error: aiError } = useAIInsights()
  const [activeTab, setActiveTab] = useState('overview')
  useSystemTheme() // This hook handles applying dark class to document

  // Extract metrics from all checks with enhanced metadata
  const allMetrics = data?.checks?.flatMap(check => 
    check.metrics?.map(metric => ({
      name: metric.name,
      value: metric.value,
      unit: metric.unit,
      labels: (metric as any).labels,
      timestamp: (metric as any).timestamp,
      type: (metric as any).type,
      checkName: check.name
    })) || []
  ) || []

  // Extract node details for enhanced visualization
  const nodeDetails = (data?.checks?.find(check => check.name === 'node-health') as any)?.details?.nodes || []
  
  // Calculate cluster summary stats
  const clusterStats = {
    totalNodes: nodeDetails.length,
    healthyNodes: nodeDetails.filter((node: any) => node.ready).length,
    avgCpuUsage: nodeDetails.reduce((sum: number, node: any) => sum + (node.cpu_percent || 0), 0) / (nodeDetails.length || 1),
    avgMemoryUsage: nodeDetails.reduce((sum: number, node: any) => sum + (node.memory_percent || 0), 0) / (nodeDetails.length || 1)
  }

  return (
    <div className="min-h-screen bg-background">
      <Header connectionStatus={connectionStatus} />
      <DashboardLayout>
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
            description={`Trend: ${(data?.score as any)?.trend || 'Unknown'} | Forecast: ${(data?.score as any)?.forecast || 'Unknown'}`}
            status="healthy"
          />
          <StatusCard
            title="Cluster Resources"
            value={`${clusterStats.healthyNodes}/${clusterStats.totalNodes}`}
            description={`Nodes Ready | Avg CPU: ${Math.round(clusterStats.avgCpuUsage)}%`}
            status="healthy"
          />
          <StatusCard
            title="AI Confidence"
            value={insights ? `${Math.round((insights.ai_confidence || 0) * 100)}%` : '--'}
            description={`Critical Issues: ${insights?.critical_issues || 0}`}
            status={(insights?.critical_issues || 0) > 0 ? 'unhealthy' : 'healthy'}
          />
        </div>

        {/* Main Dashboard Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="overview" data-state={activeTab === 'overview' ? 'active' : ''}>Overview</TabsTrigger>
            <TabsTrigger value="nodes" data-state={activeTab === 'nodes' ? 'active' : ''}>Node Details</TabsTrigger>
            <TabsTrigger value="ai-insights" data-state={activeTab === 'ai-insights' ? 'active' : ''}>AI Insights</TabsTrigger>
            <TabsTrigger value="predictions" data-state={activeTab === 'predictions' ? 'active' : ''}>Predictions</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            {/* Health Checks */}
            <HealthChecksTable 
              checks={data?.checks?.map(check => ({
                name: check.name,
                status: check.status,
                message: check.message,
                timestamp: (check as any).timestamp,
                duration: (check as any).duration
              })) || []}
            />

            {/* Enhanced Metrics */}
            <EnhancedMetricsGrid 
              metrics={allMetrics}
              clusterStats={clusterStats}
            />
          </TabsContent>

          <TabsContent value="nodes" className="space-y-6">
            <NodeDetailsPanel 
              nodes={nodeDetails}
              metrics={allMetrics.filter(m => m.labels?.node)}
            />
          </TabsContent>

          <TabsContent value="ai-insights" className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <AIInsights 
                insights={insights}
                loading={aiLoading}
                error={aiError || undefined}
              />
              <SmartAlerts />
            </div>
          </TabsContent>

          <TabsContent value="predictions" className="space-y-6">
            <PredictiveAnalytics 
              clusterHealth={data}
              insights={insights}
            />
          </TabsContent>
        </Tabs>
      </DashboardLayout>
    </div>
  )
}

export default App
