import { useCallback, useEffect, useRef, useState } from 'react'
import { config, wsUrl } from '@/config'

export interface DashboardData {
  status: "healthy" | "degraded" | "unhealthy" | "unknown"
  timestamp: string
  score?: {
    weighted: number
  }
  checks: Array<{
    name: string
    status: "healthy" | "degraded" | "unhealthy"
    message: string
    metrics?: Array<{
      name: string
      value: number
      unit?: string
    }>
  }>
}

// Validation function to check if data matches DashboardData interface
function isValidDashboardData(data: any): data is DashboardData {
  if (!data || typeof data !== 'object') {
    return false
  }

  // Check required fields
  if (!data.status || typeof data.status !== 'string') {
    return false
  }
  
  if (!data.timestamp || typeof data.timestamp !== 'string') {
    return false
  }

  // Check checks array
  if (!Array.isArray(data.checks)) {
    return false
  }

  // Validate each check structure
  for (const check of data.checks) {
    if (!check || typeof check !== 'object') {
      return false
    }
    if (!check.name || typeof check.name !== 'string' ||
        !check.status || typeof check.status !== 'string' ||
        !check.message || typeof check.message !== 'string') {
      return false
    }
  }

  return true
}

export function useWebSocket() {
  const [data, setData] = useState<DashboardData | null>(null)
  const [connectionStatus, setConnectionStatus] = useState<"connecting" | "connected" | "disconnected">("connecting")
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Extract message handler to avoid duplication
  const handleWebSocketMessage = useCallback((event: MessageEvent) => {
    try {
      // Validate that we received valid JSON
      if (!event.data || typeof event.data !== 'string') {
        console.warn('Received invalid WebSocket data format:', typeof event.data)
        return
      }

      const parsedData = JSON.parse(event.data)
      
      // Validate parsed data structure
      if (!parsedData || typeof parsedData !== 'object') {
        console.warn('Parsed WebSocket data is not a valid object:', parsedData)
        return
      }
      
      // Handle context switch events
      if (parsedData.type === 'context_switched') {
        console.log('Context switched:', parsedData.context)
        // Clear current data to show loading state
        setData(null)
      } else {
        // Validate health data structure before setting
        if (isValidDashboardData(parsedData)) {
          setData(parsedData)
        } else {
          console.warn('Received invalid dashboard data structure:', parsedData)
          // Don't update data with invalid structure
        }
      }
    } catch (error) {
      console.error('Failed to parse WebSocket data:', error, 'Raw data:', event.data)
      // Don't crash the WebSocket connection on parse errors
    }
  }, [])

  const attemptReconnect = useCallback(() => {
    if (reconnectAttemptsRef.current < config.ui.maxReconnectAttempts) {
      reconnectAttemptsRef.current++
      console.log(`Attempting to reconnect (${reconnectAttemptsRef.current}/${config.ui.maxReconnectAttempts})...`)
      
      reconnectTimeoutRef.current = setTimeout(() => {
        const websocketUrl = wsUrl()

        try {
          const ws = new WebSocket(websocketUrl)
          wsRef.current = ws

          ws.onopen = () => {
            console.log('WebSocket connected')
            setConnectionStatus('connected')
            reconnectAttemptsRef.current = 0
          }

          ws.onmessage = handleWebSocketMessage

          ws.onclose = () => {
            console.log('WebSocket disconnected')
            setConnectionStatus('disconnected')
            attemptReconnect()
          }

          ws.onerror = (error) => {
            console.error('WebSocket error:', error)
            setConnectionStatus('disconnected')
          }
        } catch (error) {
          console.error('WebSocket connection failed:', error)
          setConnectionStatus('disconnected')
        }
      }, config.ui.reconnectDelay)
    }
  }, [])

  const createWebSocket = useCallback(() => {
    const websocketUrl = wsUrl()

    try {
      const ws = new WebSocket(websocketUrl)
      wsRef.current = ws

      ws.onopen = () => {
        console.log('WebSocket connected')
        setConnectionStatus('connected')
        reconnectAttemptsRef.current = 0
      }

      ws.onmessage = handleWebSocketMessage

      ws.onclose = () => {
        console.log('WebSocket disconnected')
        setConnectionStatus('disconnected')
        attemptReconnect()
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        setConnectionStatus('disconnected')
      }
    } catch (error) {
      console.error('WebSocket connection failed:', error)
      setConnectionStatus('disconnected')
    }
  }, [attemptReconnect])


  const connect = useCallback(() => {
    createWebSocket()
  }, [createWebSocket])

  useEffect(() => {
    connect()

    const handleFocus = () => {
      if (wsRef.current?.readyState !== WebSocket.OPEN) {
        connect()
      }
    }

    const handleVisibilityChange = () => {
      if (!document.hidden && wsRef.current?.readyState !== WebSocket.OPEN) {
        connect()
      }
    }

    window.addEventListener('focus', handleFocus)
    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
      window.removeEventListener('focus', handleFocus)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [connect])

  return { data, connectionStatus }
}