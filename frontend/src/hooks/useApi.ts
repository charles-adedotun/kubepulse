import { useState, useEffect, useCallback } from 'react'
import { config, apiUrl } from '@/config'

interface UseApiOptions {
  autoFetch?: boolean
  refreshInterval?: number
  onError?: (error: Error) => void
}

export function useApi<T>(
  endpoint: string, 
  options: UseApiOptions = {}
) {
  const { 
    autoFetch = true, 
    refreshInterval = 0,
    onError 
  } = options
  
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(autoFetch)
  const [error, setError] = useState<Error | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      
      const response = await fetch(apiUrl(endpoint), {
        signal: AbortSignal.timeout(config.api.timeout),
        headers: {
          'Content-Type': 'application/json',
        },
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      const json = await response.json()
      setData(json)
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Unknown error')
      setError(error)
      onError?.(error)
    } finally {
      setLoading(false)
    }
  }, [endpoint, onError])

  useEffect(() => {
    if (autoFetch) {
      fetchData()
    }

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval)
      return () => clearInterval(interval)
    }
  }, [autoFetch, fetchData, refreshInterval])

  return { data, loading, error, refetch: fetchData }
}

// POST/PUT/DELETE operations
export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const response = await fetch(apiUrl(endpoint), {
    ...options,
    signal: AbortSignal.timeout(config.api.timeout),
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  })

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }

  return response.json()
}