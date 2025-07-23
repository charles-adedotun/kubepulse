import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { vi } from 'vitest'
import { ContextSelector } from './ContextSelector'

// Mock fetch
const mockFetch = vi.fn() as vi.MockedFunction<typeof fetch>
global.fetch = mockFetch

describe('ContextSelector', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders loading state initially', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ contexts: [] })
    } as Response)

    render(<ContextSelector />)
    
    expect(screen.getByText('Loading contexts...')).toBeInTheDocument()
    
    // Wait for the fetch to complete
    await waitFor(() => {
      expect(fetch).toHaveBeenCalled()
    })
  })

  it('displays contexts after loading', async () => {
    const mockContexts = [
      { name: 'context-1', cluster_name: 'cluster-1', namespace: 'default', current: true },
      { name: 'context-2', cluster_name: 'cluster-2', namespace: 'production', current: false }
    ]

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ contexts: mockContexts })
    } as Response)

    render(<ContextSelector />)

    await waitFor(() => {
      expect(screen.getByText('context-1')).toBeInTheDocument()
    })
  })

  it('handles context switching', async () => {
    const mockContexts = [
      { name: 'context-1', cluster_name: 'cluster-1', namespace: 'default', current: true },
      { name: 'context-2', cluster_name: 'cluster-2', namespace: 'production', current: false }
    ]

    const onContextChange = vi.fn()

    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ contexts: mockContexts })
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ 
          success: true, 
          context: { name: 'context-2', cluster_name: 'cluster-2', namespace: 'production' }
        })
      } as Response)

    render(<ContextSelector onContextChange={onContextChange} />)

    // Wait for contexts to load
    await waitFor(() => {
      expect(screen.getByText('context-1')).toBeInTheDocument()
    })

    // Click on the select to open it
    const trigger = screen.getByRole('combobox')
    fireEvent.click(trigger)

    // Select context-2
    const context2Option = await screen.findByText('context-2')
    fireEvent.click(context2Option)

    // Verify API call - fetch was called twice (initial load + switch)
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledTimes(2)
      expect(fetch).toHaveBeenNthCalledWith(
        2,
        expect.stringContaining('/api/v1/contexts/switch'),
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ context_name: 'context-2' })
        })
      )
    })

    // Verify callback was called
    expect(onContextChange).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'context-2' })
    )
  })

  it('handles fetch errors gracefully', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    
    mockFetch.mockRejectedValueOnce(new Error('Network error'))

    render(<ContextSelector />)

    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalled()
    })

    consoleSpy.mockRestore()
  })

  it('displays namespace badges', async () => {
    const mockContexts = [
      { name: 'context-1', cluster_name: 'cluster-1', namespace: 'default', current: true },
      { name: 'context-2', cluster_name: 'cluster-2', namespace: 'production', current: false }
    ]

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ contexts: mockContexts })
    } as Response)

    render(<ContextSelector />)

    await waitFor(() => {
      expect(screen.getByText('default')).toBeInTheDocument()
    })
  })

  it('disables selector while switching', async () => {
    const mockContexts = [
      { name: 'context-1', cluster_name: 'cluster-1', namespace: 'default', current: true },
      { name: 'context-2', cluster_name: 'cluster-2', namespace: 'production', current: false }
    ]

    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ contexts: mockContexts })
      } as Response)
      .mockImplementationOnce(() => new Promise(() => {})) // Never resolve to test loading state

    render(<ContextSelector />)

    await waitFor(() => {
      expect(screen.getByText('context-1')).toBeInTheDocument()
    })

    const trigger = screen.getByRole('combobox')
    fireEvent.click(trigger)

    const context2Option = await screen.findByText('context-2')
    fireEvent.click(context2Option)

    // Wait for the switching state
    await waitFor(() => {
      const updatedTrigger = screen.getByRole('combobox')
      expect(updatedTrigger).toBeDisabled()
    })
  })
})