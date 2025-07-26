// Frontend configuration with environment variable support
export interface Config {
  api: {
    baseUrl: string
    wsUrl: string
    timeout: number
  }
  ui: {
    refreshInterval: number
    aiInsightsInterval: number
    maxReconnectAttempts: number
    reconnectDelay: number
    theme: 'light' | 'dark' | 'system'
  }
  features: {
    aiInsights: boolean
    predictiveAnalytics: boolean
    smartAlerts: boolean
    nodeDetails: boolean
  }
}

// Get configuration from environment variables or defaults
function getConfig(): Config {
  // Support both Vite env vars and runtime config
  const env = import.meta.env || {}
  
  // Allow runtime configuration via window.__KUBEPULSE_CONFIG__
  const runtimeConfig = (window as Window & { __KUBEPULSE_CONFIG__?: Partial<Config> }).__KUBEPULSE_CONFIG__ || {}
  
  // Base URL can be set via env var or detected from current location
  const apiBaseUrl = runtimeConfig.apiBaseUrl || 
    env.VITE_API_BASE_URL || 
    `${window.location.protocol}//${window.location.host}`
  
  // WebSocket URL derived from API base URL
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsHost = apiBaseUrl.replace(/^https?:\/\//, '')
  const wsUrl = runtimeConfig.wsUrl || 
    env.VITE_WS_URL || 
    `${wsProtocol}//${wsHost}/ws`
  
  return {
    api: {
      baseUrl: apiBaseUrl,
      wsUrl: wsUrl,
      timeout: Number(env.VITE_API_TIMEOUT) || 30000, // 30 seconds
    },
    ui: {
      refreshInterval: Number(env.VITE_REFRESH_INTERVAL) || 10000, // 10 seconds
      aiInsightsInterval: Number(env.VITE_AI_INSIGHTS_INTERVAL) || 30000, // 30 seconds
      maxReconnectAttempts: Number(env.VITE_MAX_RECONNECT_ATTEMPTS) || 5,
      reconnectDelay: Number(env.VITE_RECONNECT_DELAY) || 3000, // 3 seconds
      theme: (env.VITE_THEME as Config['ui']['theme']) || 'system',
    },
    features: {
      aiInsights: env.VITE_FEATURE_AI_INSIGHTS !== 'false',
      predictiveAnalytics: env.VITE_FEATURE_PREDICTIVE !== 'false',
      smartAlerts: env.VITE_FEATURE_SMART_ALERTS !== 'false',
      nodeDetails: env.VITE_FEATURE_NODE_DETAILS !== 'false',
    },
  }
}

// Export singleton config
export const config = getConfig()

// Helper to build API URLs
export function apiUrl(path: string): string {
  const base = config.api.baseUrl.replace(/\/$/, '')
  const cleanPath = path.startsWith('/') ? path : `/${path}`
  return `${base}${cleanPath}`
}

// Helper to get WebSocket URL
export function wsUrl(): string {
  return config.api.wsUrl
}

// Allow dynamic config updates (useful for development)
export function updateConfig(updates: Partial<Config>): void {
  Object.assign(config, updates)
}