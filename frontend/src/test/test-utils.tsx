import React from 'react'
import { render, RenderOptions } from '@testing-library/react'
import { vi } from 'vitest'

// Mock useWebSocket hook
export const mockUseWebSocket = {
  isConnected: true,
  metrics: null,
  error: null,
  connect: vi.fn(),
  disconnect: vi.fn(),
}

// Mock useAIInsights hook
export const mockUseAIInsights = {
  insights: [],
  isLoading: false,
  error: null,
  refreshInsights: vi.fn(),
}

// Custom render function that includes providers if needed
const customRender = (
  ui: React.ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) => {
  return render(ui, {
    // Add any providers here if needed
    // wrapper: ({ children }) => <Provider>{children}</Provider>,
    ...options,
  })
}

// Helper to create event with target value
export const createChangeEvent = (value: string) => ({
  target: { value },
})

// Helper to wait for async operations
export const waitForNextTick = () => new Promise(resolve => setTimeout(resolve, 0))

// Re-export everything from testing-library
export * from '@testing-library/react'
export { customRender as render }