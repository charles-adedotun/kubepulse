import { useEffect, useState } from 'react'
import type { AIInsight } from '@/components/dashboard/AIInsights'

export function useAIInsights() {
  const [insights, setInsights] = useState<AIInsight | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchInsights = async () => {
      try {
        setLoading(true)
        const response = await fetch('/api/v1/ai/insights')
        
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
    
    // Refresh AI insights every 30 seconds
    const interval = setInterval(fetchInsights, 30000)
    
    return () => clearInterval(interval)
  }, [])

  return { insights, loading, error }
}