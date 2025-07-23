import { useEffect, useRef, useState } from 'react'

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

  const connect = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`

    try {
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        console.log('WebSocket connected')
        setConnectionStatus('connected')
        reconnectAttemptsRef.current = 0
      }

      ws.onmessage = (event) => {
        try {
          const parsedData = JSON.parse(event.data)
          setData(parsedData)
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
  }

  const attemptReconnect = () => {
    if (reconnectAttemptsRef.current < 5) {
      reconnectAttemptsRef.current++
      console.log(`Attempting to reconnect (${reconnectAttemptsRef.current}/5)...`)
      
      reconnectTimeoutRef.current = setTimeout(() => {
        connect()
      }, 3000)
    }
  }

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
  }, [])

  return { data, connectionStatus }
}