import { describe, it, expect, vi, beforeEach } from 'vitest'
import { cn } from '@/lib/utils'
import { badgeVariants } from '@/lib/badge-variants'
import { buttonVariants } from '@/lib/button-variants'

describe('UI utilities and variants', () => {
  it('merges class names and resolves Tailwind conflicts with cn()', () => {
    expect(cn('px-2 py-1', 'px-4')).toBe('py-1 px-4')
    expect(cn('text-sm', { hidden: false, active: true }, undefined, 'font-bold')).toBe('text-sm active font-bold')
  })

  it('generates expected badge variant classes', () => {
    expect(badgeVariants()).toContain('bg-primary')
    expect(badgeVariants({ variant: 'secondary' })).toContain('bg-secondary')
    expect(badgeVariants({ variant: 'destructive' })).toContain('bg-destructive')
    expect(badgeVariants({ variant: 'outline' })).toContain('text-foreground')
  })

  it('generates expected button variant and size classes', () => {
    expect(buttonVariants()).toContain('bg-primary')
    expect(buttonVariants({ variant: 'destructive', size: 'sm' })).toContain('bg-destructive')
    expect(buttonVariants({ variant: 'destructive', size: 'sm' })).toContain('h-8')
    expect(buttonVariants({ variant: 'outline', size: 'lg' })).toContain('h-10')
    expect(buttonVariants({ variant: 'ghost', size: 'icon' })).toContain('w-9')
  })
})

describe('Frontend configuration and API client', () => {
  beforeEach(() => {
    vi.stubGlobal('window', {
      location: {
        protocol: 'http:',
        host: 'localhost:8080',
      },
    })
  })

  it('builds normalized API and WebSocket URLs from config', async () => {
    const { config, apiUrl, wsUrl, updateConfig } = await import('@/config')
    expect(config.api.baseUrl).toBe('http://localhost:8080')
    expect(apiUrl('/api/v1/health')).toBe('http://localhost:8080/api/v1/health')
    expect(apiUrl('api/v1/health')).toBe('http://localhost:8080/api/v1/health')
    expect(wsUrl()).toBe('ws://localhost:8080/ws')

    updateConfig({
      api: {
        baseUrl: 'https://kubepulse.example.com/',
        wsUrl: 'wss://kubepulse.example.com/ws',
        timeout: 15000,
      },
    })
    expect(apiUrl('/api/v1/metrics')).toBe('https://kubepulse.example.com/api/v1/metrics')
    expect(wsUrl()).toBe('wss://kubepulse.example.com/ws')
  })

  it('handles successful responses and HTTP errors in apiRequest', async () => {
    const { apiRequest } = await import('@/hooks/useApi')

    const mockFetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ status: 'healthy' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const result = await apiRequest<{ status: string }>('/api/v1/health')
    expect(result).toEqual({ status: 'healthy' })
    expect(mockFetch).toHaveBeenCalledTimes(1)

    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 503,
    })

    await expect(apiRequest('/api/v1/health')).rejects.toThrow('HTTP error! status: 503')
  })
})
