import { createContext, useContext, useEffect, useState, useCallback } from 'react'
import type { ReactNode } from 'react'
import { apiUrl } from '@/config'

export interface ConnectionStatus {
  status: 'connected' | 'disconnected' | 'no_contexts' | 'invalid_context' | 'loading'
  hasContexts: boolean
  current?: {
    name: string
    cluster_name: string
    namespace: string
    server: string
    user: string
    current: boolean
  }
  error?: string
  message: string
  canRetry: boolean
  suggestions?: string[]
  details?: Record<string, string>
}

interface ConnectionStatusContextType {
  connectionStatus: ConnectionStatus
  refreshStatus: () => Promise<void>
  isRefreshing: boolean
  lastChecked?: Date
}

const ConnectionStatusContext = createContext<ConnectionStatusContextType | undefined>(undefined)

interface ConnectionStatusProviderProps {
  children: ReactNode
  checkInterval?: number
  autoRefresh?: boolean
}

export function ConnectionStatusProvider({ 
  children, 
  checkInterval = 30000, // 30 seconds
  autoRefresh = true 
}: ConnectionStatusProviderProps) {
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>({
    status: 'loading',
    hasContexts: false,
    message: 'Checking connection status...',
    canRetry: false
  })
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [lastChecked, setLastChecked] = useState<Date>()

  const refreshStatus = useCallback(async () => {
    if (isRefreshing) return

    setIsRefreshing(true)
    try {
      const response = await fetch(apiUrl('/api/v1/contexts/status'), {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        throw new Error(`API request failed: ${response.status} ${response.statusText}`)
      }

      const status: ConnectionStatus = await response.json()
      setConnectionStatus(status)
      setLastChecked(new Date())
    } catch (error) {
      console.error('Failed to check connection status:', error)
      setConnectionStatus({
        status: 'disconnected',
        hasContexts: false,
        message: 'Unable to connect to KubePulse server',
        canRetry: true,
        error: error instanceof Error ? error.message : 'Unknown error',
        suggestions: [
          'Check if the KubePulse server is running',
          'Verify network connectivity',
          'Refresh the page',
        ]
      })
    } finally {
      setIsRefreshing(false)
    }
  }, [isRefreshing])

  // Initial check
  useEffect(() => {
    refreshStatus()
  }, [refreshStatus])

  // Auto-refresh interval
  useEffect(() => {
    if (!autoRefresh) return

    const interval = setInterval(() => {
      // Only auto-refresh if we're not already refreshing and the last check was successful
      if (!isRefreshing && connectionStatus.status !== 'loading') {
        refreshStatus()
      }
    }, checkInterval)

    return () => clearInterval(interval)
  }, [autoRefresh, checkInterval, isRefreshing, connectionStatus.status, refreshStatus])

  // Refresh on window focus (useful when user returns to tab)
  useEffect(() => {
    const handleFocus = () => {
      if (!isRefreshing) {
        refreshStatus()
      }
    }

    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [isRefreshing, refreshStatus])

  // Refresh on network state change
  useEffect(() => {
    const handleOnline = () => {
      if (!isRefreshing) {
        refreshStatus()
      }
    }

    window.addEventListener('online', handleOnline)
    return () => window.removeEventListener('online', handleOnline)
  }, [isRefreshing, refreshStatus])

  const value: ConnectionStatusContextType = {
    connectionStatus,
    refreshStatus,
    isRefreshing,
    lastChecked
  }

  return (
    <ConnectionStatusContext.Provider value={value}>
      {children}
    </ConnectionStatusContext.Provider>
  )
}

export function useConnectionStatus() {
  const context = useContext(ConnectionStatusContext)
  if (context === undefined) {
    throw new Error('useConnectionStatus must be used within a ConnectionStatusProvider')
  }
  return context
}

// Hook to check if the dashboard should be fully rendered
export function useShouldRenderDashboard() {
  const { connectionStatus } = useConnectionStatus()
  
  // Only render full dashboard when we have a working connection
  return connectionStatus.status === 'connected'
}

// Hook to get appropriate loading/error state for components
export function useComponentState() {
  const { connectionStatus, isRefreshing } = useConnectionStatus()
  
  return {
    isLoading: connectionStatus.status === 'loading' || isRefreshing,
    hasError: ['disconnected', 'no_contexts', 'invalid_context'].includes(connectionStatus.status),
    isConnected: connectionStatus.status === 'connected',
    error: connectionStatus.error,
    message: connectionStatus.message,
    suggestions: connectionStatus.suggestions || [],
    canRetry: connectionStatus.canRetry
  }
}