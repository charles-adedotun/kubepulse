import { useEffect, useState } from 'react'
import type { AIInsight } from '@/components/dashboard/AIInsights'
import { config, apiUrl } from '@/config'

export function useAIInsights() {
  const [insights, setInsights] = useState<AIInsight | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchInsights = async () => {
      try {
        setLoading(true)
        const response = await fetch(apiUrl('/api/v1/ai/insights'), {
          signal: AbortSignal.timeout(config.api.timeout)
        })
        
        if (!response.ok) {
          throw new Error('Failed to fetch AI insights')
        }
        
        const data = await response.json()
        setInsights(data)
        setError(null)
      } catch (err) {
        console.error('Failed to load AI insights:', err)
        setError(err instanceof Error ? err.message : 'Unknown error')
        setInsights(null)
      } finally {
        setLoading(false)
      }
    }

    fetchInsights()
    
    // Refresh AI insights based on config
    const interval = setInterval(fetchInsights, config.ui.aiInsightsInterval)
    
    return () => clearInterval(interval)
  }, [])

  return { insights, loading, error }
}