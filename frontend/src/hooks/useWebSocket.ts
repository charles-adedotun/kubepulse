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

export function useWebSocket() {
  const [data, setData] = useState<DashboardData | null>(null)
  const [connectionStatus, setConnectionStatus] = useState<"connecting" | "connected" | "disconnected">("connecting")
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

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

          ws.onmessage = (event) => {
            try {
              const parsedData = JSON.parse(event.data)
              
              // Handle context switch events
              if (parsedData.type === 'context_switched') {
                console.log('Context switched:', parsedData.context)
                // Clear current data to show loading state
                setData(null)
              } else {
                // Regular health data update
                setData(parsedData)
              }
            } catch (error) {
              console.error('Failed to parse WebSocket data:', error)
            }
          }

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

      ws.onmessage = (event) => {
        try {
          const parsedData = JSON.parse(event.data)
          
          // Handle context switch events
          if (parsedData.type === 'context_switched') {
            console.log('Context switched:', parsedData.context)
            // Clear current data to show loading state
            setData(null)
          } else {
            // Regular health data update
            setData(parsedData)
          }
        } catch (error) {
          console.error('Failed to parse WebSocket data:', error)
        }
      }

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