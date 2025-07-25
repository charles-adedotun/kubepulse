import { useEffect, useState } from 'react'
import type { AIInsight } from '@/components/dashboard/AIInsights'
import { config, apiUrl } from '@/config'

// Type guard to validate AI insight response
function isValidAIInsight(data: any): data is AIInsight {
  if (!data || typeof data !== 'object') {
    return false
  }

  // Check required fields
  if (typeof data.overall_health !== 'string' ||
      typeof data.ai_confidence !== 'number' ||
      typeof data.critical_issues !== 'number') {
    return false
  }

  // Validate optional fields if present
  if (data.trend_analysis && typeof data.trend_analysis !== 'string') {
    return false
  }

  if (data.predicted_issues && !Array.isArray(data.predicted_issues)) {
    return false
  }

  if (data.top_recommendations) {
    if (!Array.isArray(data.top_recommendations)) {
      return false
    }
    // Validate recommendation structure
    for (const rec of data.top_recommendations) {
      if (!rec || typeof rec !== 'object' ||
          typeof rec.title !== 'string' ||
          typeof rec.description !== 'string') {
        return false
      }
    }
  }

  return true
}

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
        
        // Validate response structure before using it
        if (isValidAIInsight(data)) {
          setInsights(data)
          setError(null)
        } else {
          console.warn('Received invalid AI insight data structure:', data)
          setError('Invalid data format received from server')
          setInsights(null)
        }
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