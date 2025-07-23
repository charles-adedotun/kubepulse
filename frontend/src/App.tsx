import { Header } from '@/components/layout/Header'
import { DashboardLayout } from '@/components/layout/DashboardLayout'
import { StatusCard } from '@/components/dashboard/StatusCard'
import { HealthChecksTable } from '@/components/dashboard/HealthChecksTable'
import { MetricsGrid } from '@/components/dashboard/MetricsGrid'
import { AIInsights } from '@/components/dashboard/AIInsights'
import { useWebSocket } from '@/hooks/useWebSocket'
import { useAIInsights } from '@/hooks/useAIInsights'
import { useSystemTheme } from '@/hooks/useSystemTheme'

function App() {
  const { data, connectionStatus } = useWebSocket()
  const { insights, loading: aiLoading, error: aiError } = useAIInsights()
  useSystemTheme() // This hook handles applying dark class to document

  // Extract metrics from all checks
  const allMetrics = data?.checks?.flatMap(check => 
    check.metrics?.map(metric => ({
      name: metric.name,
      value: metric.value,
      unit: metric.unit
    })) || []
  ) || []

  return (
    <div className="min-h-screen bg-background">
      <Header connectionStatus={connectionStatus} />
      <DashboardLayout>
        {/* Status Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <StatusCard
            title="Overall Health"
            value={data?.status?.toUpperCase() || 'UNKNOWN'}
            description={data ? `Last updated: ${new Date(data.timestamp).toLocaleTimeString()}` : 'Loading...'}
            status={data?.status || 'unknown'}
          />
          <StatusCard
            title="Health Score"
            value={data?.score ? `${Math.round(data.score.weighted)}%` : '--'}
            description="Weighted ML confidence score"
            status="healthy"
          />
          <StatusCard
            title="Active Checks"
            value={data?.checks?.length || 0}
            description="Real-time monitoring active"
            status="healthy"
          />
        </div>

        {/* Health Checks */}
        <div className="mb-8">
          <HealthChecksTable 
            checks={data?.checks?.map(check => ({
              name: check.name,
              status: check.status,
              message: check.message
            })) || []}
          />
        </div>

        {/* Metrics */}
        <div className="mb-8">
          <MetricsGrid metrics={allMetrics} />
        </div>

        {/* AI Insights */}
        <div>
          <AIInsights 
            insights={insights}
            loading={aiLoading}
            error={aiError || undefined}
          />
        </div>
      </DashboardLayout>
    </div>
  )
}

export default App
