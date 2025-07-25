import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { StatusCard } from './StatusCard'

describe('StatusCard', () => {
  const defaultProps = {
    title: 'Test Status',
    value: '100%',
    description: 'Test description',
  }

  it('renders title, value, and description', () => {
    render(<StatusCard {...defaultProps} />)
    
    expect(screen.getByText('Test Status')).toBeInTheDocument()
    expect(screen.getByText('100%')).toBeInTheDocument()
    expect(screen.getByText('Test description')).toBeInTheDocument()
  })

  it('displays numeric values correctly', () => {
    render(<StatusCard {...defaultProps} value={42} />)
    
    expect(screen.getByText('42')).toBeInTheDocument()
  })

  it('applies correct status colors', () => {
    const { rerender } = render(<StatusCard {...defaultProps} status="healthy" />)
    let statusIndicator = document.querySelector('.bg-green-500')
    expect(statusIndicator).toBeInTheDocument()

    rerender(<StatusCard {...defaultProps} status="degraded" />)
    statusIndicator = document.querySelector('.bg-yellow-500')
    expect(statusIndicator).toBeInTheDocument()

    rerender(<StatusCard {...defaultProps} status="unhealthy" />)
    statusIndicator = document.querySelector('.bg-red-500')
    expect(statusIndicator).toBeInTheDocument()

    rerender(<StatusCard {...defaultProps} status="unknown" />)
    statusIndicator = document.querySelector('.bg-gray-500')
    expect(statusIndicator).toBeInTheDocument()
  })

  it('defaults to unknown status when status prop is not provided', () => {
    render(<StatusCard {...defaultProps} />)
    
    const statusIndicator = document.querySelector('.bg-gray-500')
    expect(statusIndicator).toBeInTheDocument()
  })

  it('applies custom className when provided', () => {
    render(<StatusCard {...defaultProps} className="custom-class" />)
    
    const card = document.querySelector('.custom-class')
    expect(card).toBeInTheDocument()
  })

  it('has proper semantic structure', () => {
    render(<StatusCard {...defaultProps} />)
    
    // Check that the title is properly structured
    const title = screen.getByText('Test Status')
    expect(title).toHaveClass('text-base', 'font-semibold')
    
    // Check that the value has proper styling
    const value = screen.getByText('100%')
    expect(value).toHaveClass('text-lg', 'font-semibold')
    
    // Check that the description has proper styling
    const description = screen.getByText('Test description')
    expect(description).toHaveClass('text-sm', 'text-muted-foreground')
  })

  it('renders status indicator with correct positioning', () => {
    render(<StatusCard {...defaultProps} status="healthy" />)
    
    const statusIndicator = document.querySelector('.h-2.w-2.rounded-full.bg-green-500')
    expect(statusIndicator).toBeInTheDocument()
  })

  it('handles long text content properly', () => {
    const longProps = {
      title: 'Very Long Status Title That Might Wrap',
      value: '99.99999%',
      description: 'This is a very long description that might wrap to multiple lines and should still display properly within the card component',
    }
    
    render(<StatusCard {...longProps} />)
    
    expect(screen.getByText(longProps.title)).toBeInTheDocument()
    expect(screen.getByText(longProps.value)).toBeInTheDocument()
    expect(screen.getByText(longProps.description)).toBeInTheDocument()
  })
})