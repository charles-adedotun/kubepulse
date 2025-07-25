import { vi } from 'vitest'

// Mock WebSocket
export const mockWebSocket = {
  send: vi.fn(),
  close: vi.fn(),
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  readyState: WebSocket.OPEN,
}

// Mock fetch responses
export const mockFetchResponse = (data: any, ok = true, status = 200) => {
  return Promise.resolve({
    ok,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  })
}

// Common mock data
export const mockMetricsData = {
  cpu: { value: 45.2, unit: '%', trend: 'up' as const },
  memory: { value: 2.8, unit: 'GB', trend: 'stable' as const },
  network: { value: 125.4, unit: 'MB/s', trend: 'down' as const },
  pods: { value: 42, unit: 'running', trend: 'up' as const },
}

export const mockNodeData = {
  name: 'node-1',
  status: 'Ready',
  cpu: 45.2,
  memory: 78.5,
  pods: 12,
  conditions: [
    { type: 'Ready', status: 'True', reason: 'KubeletReady' },
    { type: 'DiskPressure', status: 'False', reason: 'KubeletHasNoDiskPressure' },
  ],
}

export const mockAIInsights = {
  insights: [
    {
      id: '1',
      type: 'performance',
      severity: 'medium',
      title: 'High CPU Usage Detected',
      description: 'Node node-1 is experiencing high CPU usage',
      recommendations: ['Scale up the cluster', 'Optimize workloads'],
    },
  ],
  isLoading: false,
  error: null,
}

// Setup function to mock global fetch
export const setupFetchMock = () => {
  global.fetch = vi.fn()
  return global.fetch as any
}

// Setup function to mock WebSocket
export const setupWebSocketMock = () => {
  global.WebSocket = vi.fn(() => mockWebSocket) as any
  return mockWebSocket
}

// Cleanup function
export const cleanupMocks = () => {
  vi.restoreAllMocks()
}