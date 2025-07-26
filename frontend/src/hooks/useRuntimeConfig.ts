import { useEffect } from 'react'
import { config, updateConfig, apiUrl } from '@/config'

// This hook fetches runtime configuration from the server
// and updates the local config with server-provided values
export function useRuntimeConfig() {
  useEffect(() => {
    const fetchRuntimeConfig = async () => {
      try {
        const response = await fetch(apiUrl('/api/v1/config/ui'))
        
        if (!response.ok) {
          console.warn('Failed to fetch runtime config, using defaults')
          return
        }
        
        const runtimeConfig = await response.json()
        
        // Update the config with server values
        updateConfig({
          ui: {
            ...config.ui,
            refreshInterval: runtimeConfig.refreshInterval || config.ui.refreshInterval,
            aiInsightsInterval: runtimeConfig.aiInsightsInterval || config.ui.aiInsightsInterval,
            maxReconnectAttempts: runtimeConfig.maxReconnectAttempts || config.ui.maxReconnectAttempts,
            reconnectDelay: runtimeConfig.reconnectDelay || config.ui.reconnectDelay,
            theme: runtimeConfig.theme || config.ui.theme,
          },
          features: {
            ...config.features,
            ...runtimeConfig.features,
          },
        })
        
        console.log('Runtime configuration loaded', runtimeConfig)
      } catch (error) {
        console.error('Failed to load runtime config:', error)
      }
    }
    
    // Fetch config immediately
    fetchRuntimeConfig()
  }, [])
}