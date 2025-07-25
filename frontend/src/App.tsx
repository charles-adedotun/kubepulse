import { Header } from '@/components/layout/Header'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { AIEnhancedStatusCard, generateMockAIAnalysis } from '@/components/dashboard/AIEnhancedStatusCard'
import { AIKeyInsights } from '@/components/dashboard/AIKeyInsights'
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
import { config } from '@/config'
import { useRuntimeConfig } from '@/hooks/useRuntimeConfig'
import { ConnectionStatusProvider, useConnectionStatus } from '@/contexts/ConnectionStatusContext'
import { EmptyStateCard, LoadingStateCard, NoContextsCard, DisconnectedCard, InvalidContextCard } from '@/components/ui/empty-state'
import type { ContextInfo } from '@/components/dashboard/ContextSelector'
import type { EnhancedMetric } from '@/components/dashboard/EnhancedMetricsGrid'
import type { DashboardData } from '@/hooks/useWebSocket'

// Extended metric type with enhanced metadata
interface ExtendedMetric extends EnhancedMetric {
  labels?: Record<string, string>
  timestamp?: string
  type?: string
  checkName?: string
}

// Node detail type
interface NodeDetail {
  name: string
  ready: boolean
  cpu_allocatable: number
  cpu_percent: number
  memory_allocatable: number
  memory_percent: number
}

// Cluster statistics type
interface ClusterStats {
  totalNodes: number
  healthyNodes: number
  avgCpuUsage: number
  avgMemoryUsage: number
}

// Score with extended properties
interface ExtendedScore {
  weighted: number
  trend?: string
  forecast?: string
}

// Smart Dashboard component that renders based on connection status
function SmartDashboard() {
  const { data, connectionStatus } = useWebSocket()
  const { insights, loading: aiLoading, error: aiError } = useAIInsights()
  const [activeTab, setActiveTab] = useState('overview')
  const { connectionStatus: detailedStatus, refreshStatus, isRefreshing } = useConnectionStatus()

  const handleContextChange = (_context: ContextInfo) => {
    // The websocket connection will automatically receive updates
    // for the new context from the server
  }

  // Extract metrics from all checks with enhanced metadata
  const allMetrics: ExtendedMetric[] = data?.checks?.flatMap(check => 
    check.metrics?.map(metric => ({
      name: metric.name,
      value: metric.value,
      unit: metric.unit,
      labels: (metric as ExtendedMetric).labels,
      timestamp: (metric as ExtendedMetric).timestamp,
      type: (metric as ExtendedMetric).type,
      checkName: check.name
    })) || []
  ) || []

  // Extract node details for enhanced visualization
  const nodeHealthCheck = data?.checks?.find(check => check.name === 'node-health') as DashboardData['checks'][0] & { details?: { nodes: NodeDetail[] } }
  const nodeDetails: NodeDetail[] = nodeHealthCheck?.details?.nodes || []
  
  // Calculate cluster summary stats
  const clusterStats: ClusterStats = {
    totalNodes: nodeDetails.length,
    healthyNodes: nodeDetails.filter((node: NodeDetail) => node.ready).length,
    avgCpuUsage: nodeDetails.reduce((sum: number, node: NodeDetail) => sum + (node.cpu_percent || 0), 0) / (nodeDetails.length || 1),
    avgMemoryUsage: nodeDetails.reduce((sum: number, node: NodeDetail) => sum + (node.memory_percent || 0), 0) / (nodeDetails.length || 1)
  }

  // Render appropriate state based on connection status
  const renderContent = () => {
    switch (detailedStatus.status) {
      case 'loading':
        return (
          <div className="flex justify-center items-center min-h-[400px]">
            <LoadingStateCard 
              title="Checking Connection"
              message="Verifying Kubernetes connectivity..."
            />
          </div>
        )

      case 'no_contexts':
        return (
          <div className="flex justify-center items-center min-h-[400px]">
            <NoContextsCard
              error={detailedStatus.error}
              message={detailedStatus.message}
              suggestions={detailedStatus.suggestions}
              canRetry={detailedStatus.canRetry}
              onRetry={refreshStatus}
              isRetrying={isRefreshing}
            />
          </div>
        )

      case 'invalid_context':
        return (
          <div className="flex justify-center items-center min-h-[400px]">
            <InvalidContextCard
              error={detailedStatus.error}
              message={detailedStatus.message}
              suggestions={detailedStatus.suggestions}
              canRetry={detailedStatus.canRetry}
              onRetry={refreshStatus}
              isRetrying={isRefreshing}
            />
          </div>
        )

      case 'disconnected':
        return (
          <div className="flex justify-center items-center min-h-[400px]">
            <DisconnectedCard
              error={detailedStatus.error}
              message={detailedStatus.message}
              suggestions={detailedStatus.suggestions}
              canRetry={detailedStatus.canRetry}
              onRetry={refreshStatus}
              isRetrying={isRefreshing}
            />
          </div>
        )

      case 'connected':
        // Render full dashboard
        return (
          <>
            {/* Context Info Bar */}
            {detailedStatus.current && (
              <div className="mb-3 p-2 bg-muted/50 rounded-lg border">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-normal">Context:</span>
                  <span className="text-sm font-semibold">{detailedStatus.current.name}</span>
                  <span className="text-sm text-muted-foreground">•</span>
                  <span className="text-sm text-muted-foreground">{detailedStatus.current.cluster_name}</span>
                </div>
              </div>
            )}

            {/* AI-Enhanced Status Cards */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-3 mb-4">
              <AIEnhancedStatusCard
                title="Overall Health"
                value={data?.status?.toUpperCase() || 'UNKNOWN'}
                description={data ? `Last updated: ${new Date(data.timestamp).toLocaleTimeString()}` : 'Loading...'}
                status={data?.status || 'unknown'}
                aiAnalysis={generateMockAIAnalysis('health')}
              />
              <AIEnhancedStatusCard
                title="Health Score"
                value={data?.score ? `${Math.round(data.score.weighted)}%` : '--'}
                description={`Trend: ${(data?.score as ExtendedScore)?.trend || 'Unknown'} | Forecast: ${(data?.score as ExtendedScore)?.forecast || 'Unknown'}`}
                status="healthy"
                aiAnalysis={generateMockAIAnalysis('score')}
              />
              <AIEnhancedStatusCard
                title="Cluster Resources"
                value={`${clusterStats.healthyNodes}/${clusterStats.totalNodes}`}
                description={`Nodes Ready | Avg CPU: ${Math.round(clusterStats.avgCpuUsage)}%`}
                status="healthy"
                aiAnalysis={generateMockAIAnalysis('resources')}
              />
              {config.ui.features.aiInsights && (
                <AIEnhancedStatusCard
                  title="AI Confidence"
                  value={insights ? `${Math.round((insights.ai_confidence || 0) * 100)}%` : '--'}
                  description={`Critical Issues: ${insights?.critical_issues || 0}`}
                  status={(insights?.critical_issues || 0) > 0 ? 'unhealthy' : 'healthy'}
                  aiAnalysis={generateMockAIAnalysis('confidence')}
                />
              )}
            </div>

            {/* Dashboard Tabs */}
            <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-3">
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="nodes">Node Details</TabsTrigger>
                <TabsTrigger value="ai-insights">AI Insights</TabsTrigger>
                <TabsTrigger value="predictions">Predictions</TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="space-y-6">
                {/* AI Key Insights - prominently featured */}
                {config.ui.features.aiInsights && (
                  <AIKeyInsights 
                    clusterName={detailedStatus.current?.cluster_name}
                    loading={!data}
                  />
                )}

                <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
                  {/* Health Checks */}
                  <div className="xl:col-span-2">
                    <HealthChecksTable 
                      checks={data?.checks?.map(check => ({
                        name: check.name,
                        status: check.status,
                        message: check.message,
                        timestamp: (check as DashboardData['checks'][0] & { timestamp?: string }).timestamp,
                        duration: (check as DashboardData['checks'][0] & { duration?: number }).duration
                      })) || []}
                    />
                  </div>

                  {/* Enhanced Metrics */}
                  <div className="xl:col-span-1">
                    <EnhancedMetricsGrid 
                      metrics={allMetrics}
                      clusterStats={clusterStats}
                    />
                  </div>
                </div>
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
                  clusterHealth={data || undefined}
                  insights={insights || undefined}
                />
              </TabsContent>
            </Tabs>
          </>
        )

      default:
        return (
          <div className="flex justify-center items-center min-h-[400px]">
            <EmptyStateCard
              state="no_data"
              message="Unknown connection state"
              canRetry={true}
              onRetry={refreshStatus}
              isRetrying={isRefreshing}
            />
          </div>
        )
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <Header connectionStatus={connectionStatus} onContextChange={handleContextChange} />
      <DashboardLayout>
        {renderContent()}
      </DashboardLayout>
    </div>
  )
}

function App() {
  // Load runtime configuration from server
  useRuntimeConfig()
  useSystemTheme() // This hook handles applying dark class to document

  return (
    <ConnectionStatusProvider>
      <SmartDashboard />
    </ConnectionStatusProvider>
  )
}

export default App
