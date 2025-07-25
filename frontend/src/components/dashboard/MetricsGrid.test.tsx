import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { MetricsGrid, type Metric } from './MetricsGrid'

describe('MetricsGrid', () => {
  const mockMetrics: Metric[] = [
    { name: 'cpu_usage', value: 75.5, unit: '%' },
    { name: 'memory_usage', value: 4.2, unit: 'GB' },
    { name: 'network_io', value: 125, unit: 'MB/s' },
    { name: 'active_pods', value: 42 },
  ]

  it('renders metrics grid with correct title', () => {
    render(<MetricsGrid metrics={mockMetrics} />)
    
    expect(screen.getByText('📊')).toBeInTheDocument()
    expect(screen.getByText('Key Metrics')).toBeInTheDocument()
  })

  it('displays all provided metrics', () => {
    render(<MetricsGrid metrics={mockMetrics} />)
    
    expect(screen.getByText('75.5')).toBeInTheDocument()
    expect(screen.getByText('%')).toBeInTheDocument()
    expect(screen.getByText('4.2')).toBeInTheDocument()
    expect(screen.getByText('GB')).toBeInTheDocument()
    expect(screen.getByText('125')).toBeInTheDocument()
    expect(screen.getByText('MB/s')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('formats metric names correctly', () => {
    render(<MetricsGrid metrics={mockMetrics} />)
    
    expect(screen.getByText('cpu usage')).toBeInTheDocument()
    expect(screen.getByText('memory usage')).toBeInTheDocument()
    expect(screen.getByText('network io')).toBeInTheDocument()
    expect(screen.getByText('active pods')).toBeInTheDocument()
  })

  it('handles metrics without units', () => {
    const metricsWithoutUnit: Metric[] = [
      { name: 'count', value: 100 }
    ]
    
    render(<MetricsGrid metrics={metricsWithoutUnit} />)
    
    expect(screen.getByText('100')).toBeInTheDocument()
    expect(screen.getByText('count')).toBeInTheDocument()
  })

  it('displays empty state when no metrics provided', () => {
    render(<MetricsGrid metrics={[]} />)
    
    expect(screen.getByText('Key Metrics')).toBeInTheDocument()
    expect(screen.getByText('No metrics available')).toBeInTheDocument()
  })

  it('limits display to maximum of 8 metrics', () => {
    const manyMetrics: Metric[] = Array.from({ length: 12 }, (_, i) => ({
      name: `metric_${i}`,
      value: i * 10,
      unit: 'unit'
    }))
    
    render(<MetricsGrid metrics={manyMetrics} />)
    
    // Should display first 8 metrics
    expect(screen.getByText('metric 0')).toBeInTheDocument()
    expect(screen.getByText('metric 7')).toBeInTheDocument()
    
    // Should not display 9th metric and beyond
    expect(screen.queryByText('metric 8')).not.toBeInTheDocument()
    expect(screen.queryByText('metric 11')).not.toBeInTheDocument()
  })

  it('applies correct CSS classes for responsive grid', () => {
    render(<MetricsGrid metrics={mockMetrics} />)
    
    const gridContainer = document.querySelector('.grid.grid-cols-2.md\\:grid-cols-3.lg\\:grid-cols-4.gap-4')
    expect(gridContainer).toBeInTheDocument()
  })

  it('applies hover effects to metric cards', () => {
    render(<MetricsGrid metrics={mockMetrics} />)
    
    const metricCard = document.querySelector('.hover\\:bg-secondary\\/70')
    expect(metricCard).toBeInTheDocument()
  })

  it('handles decimal values correctly', () => {
    const decimalMetrics: Metric[] = [
      { name: 'cpu', value: 75.567, unit: '%' }
    ]
    
    render(<MetricsGrid metrics={decimalMetrics} />)
    
    expect(screen.getByText('75.567')).toBeInTheDocument()
  })

  it('handles zero values', () => {
    const zeroMetrics: Metric[] = [
      { name: 'errors', value: 0, unit: 'count' }
    ]
    
    render(<MetricsGrid metrics={zeroMetrics} />)
    
    expect(screen.getByText('0')).toBeInTheDocument()
    expect(screen.getByText('errors')).toBeInTheDocument()
  })

  it('handles negative values', () => {
    const negativeMetrics: Metric[] = [
      { name: 'change', value: -15.5, unit: '%' }
    ]
    
    render(<MetricsGrid metrics={negativeMetrics} />)
    
    expect(screen.getByText('-15.5')).toBeInTheDocument()
  })

  it('uses correct typography classes', () => {
    render(<MetricsGrid metrics={mockMetrics} />)
    
    // Check that elements with correct classes exist
    const valueContainer = document.querySelector('.text-xl.font-semibold.text-primary')
    expect(valueContainer).toBeInTheDocument()
    expect(valueContainer).toHaveTextContent('75.5')
    
    // Check name styling
    const nameElement = screen.getByText('cpu usage')
    expect(nameElement).toHaveClass('text-sm', 'text-muted-foreground', 'mt-1', 'uppercase')
  })
})