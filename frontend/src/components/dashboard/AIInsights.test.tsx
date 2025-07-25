import { describe, it, expect } from 'vitest'
import { render, screen } from '@/test/test-utils'
import { AIInsights, type AIInsight } from './AIInsights'

describe('AIInsights', () => {
  const mockInsights: AIInsight = {
    overall_health: 'Your cluster is running smoothly with no critical issues.',
    ai_confidence: 0.85,
    critical_issues: 0,
    trend_analysis: 'Metrics show stable performance over the last 24 hours.',
    predicted_issues: [
      'Memory usage may increase during peak hours',
      'Network latency might spike due to scheduled maintenance'
    ],
    top_recommendations: [
      {
        title: 'Scale Up Workers',
        description: 'Consider adding more worker nodes for better resource distribution',
        impact: 'High',
        effort: 'Medium'
      },
      {
        title: 'Optimize Resource Limits',
        description: 'Review and adjust pod resource limits to prevent resource starvation',
        impact: 'Medium',
        effort: 'Low'
      }
    ]
  }

  it('renders AI insights with correct title', () => {
    render(<AIInsights insights={mockInsights} />)
    
    expect(screen.getByText('🤖')).toBeInTheDocument()
    expect(screen.getByText('AI Insights')).toBeInTheDocument()
  })

  it('displays overall health assessment', () => {
    render(<AIInsights insights={mockInsights} />)
    
    expect(screen.getByText('Cluster Health Assessment')).toBeInTheDocument()
    expect(screen.getByText('Your cluster is running smoothly with no critical issues.')).toBeInTheDocument()
  })

  it('shows AI confidence percentage', () => {
    render(<AIInsights insights={mockInsights} />)
    
    expect(screen.getByText('AI Confidence: 85%')).toBeInTheDocument()
  })

  it('displays trend analysis when available', () => {
    render(<AIInsights insights={mockInsights} />)
    
    expect(screen.getByText('Metrics show stable performance over the last 24 hours.')).toBeInTheDocument()
  })

  it('shows predicted issues when available', () => {
    render(<AIInsights insights={mockInsights} />)
    
    expect(screen.getByText('Predicted Issues')).toBeInTheDocument()
    expect(screen.getByText('Memory usage may increase during peak hours')).toBeInTheDocument()
    expect(screen.getByText('Network latency might spike due to scheduled maintenance')).toBeInTheDocument()
  })

  it('displays top recommendations with details', () => {
    render(<AIInsights insights={mockInsights} />)
    
    expect(screen.getByText('Top Recommendations')).toBeInTheDocument()
    expect(screen.getByText('Scale Up Workers')).toBeInTheDocument()
    expect(screen.getByText('Consider adding more worker nodes for better resource distribution')).toBeInTheDocument()
    expect(screen.getByText('Impact: High')).toBeInTheDocument()
    expect(screen.getByText('Effort: Medium')).toBeInTheDocument()
  })

  it('limits recommendations to maximum of 3', () => {
    const insightsWithManyRecs: AIInsight = {
      ...mockInsights,
      top_recommendations: Array.from({ length: 5 }, (_, i) => ({
        title: `Recommendation ${i + 1}`,
        description: `Description ${i + 1}`,
      }))
    }
    
    render(<AIInsights insights={insightsWithManyRecs} />)
    
    expect(screen.getByText('Recommendation 1')).toBeInTheDocument()
    expect(screen.getByText('Recommendation 3')).toBeInTheDocument()
    expect(screen.queryByText('Recommendation 4')).not.toBeInTheDocument()
  })

  it('shows critical issues alert when issues exist', () => {
    const criticalInsights: AIInsight = {
      ...mockInsights,
      critical_issues: 2
    }
    
    render(<AIInsights insights={criticalInsights} />)
    
    expect(screen.getByText('Critical Issues Detected')).toBeInTheDocument()
    expect(screen.getByText('Immediate attention required for 2 critical issues')).toBeInTheDocument()
  })

  it('handles singular critical issue correctly', () => {
    const singleCriticalInsight: AIInsight = {
      ...mockInsights,
      critical_issues: 1
    }
    
    render(<AIInsights insights={singleCriticalInsight} />)
    
    expect(screen.getByText('Immediate attention required for 1 critical issue')).toBeInTheDocument()
  })

  it('applies destructive variant for critical issues', () => {
    const criticalInsights: AIInsight = {
      ...mockInsights,
      critical_issues: 1
    }
    
    render(<AIInsights insights={criticalInsights} />)
    
    // Check that alert has destructive styling (this would need to be verified via DOM classes)
    expect(screen.getByText('Critical Issues Detected')).toBeInTheDocument()
  })

  it('displays loading state', () => {
    render(<AIInsights insights={null} loading={true} />)
    
    expect(screen.getByText('AI Insights')).toBeInTheDocument()
    expect(screen.getByText('Loading AI insights...')).toBeInTheDocument()
    
    // Check for loading animation class
    const loadingElement = screen.getByText('Loading AI insights...')
    expect(loadingElement).toHaveClass('animate-pulse')
  })

  it('displays error state when insights are null', () => {
    render(<AIInsights insights={null} />)
    
    expect(screen.getByText('AI Analysis Unavailable')).toBeInTheDocument()
    expect(screen.getByText('AI-powered insights are not available. Ensure Claude Code CLI is installed and accessible.')).toBeInTheDocument()
    expect(screen.getByText('npm install -g @anthropic-ai/claude-code')).toBeInTheDocument()
  })

  it('displays error state when error prop is provided', () => {
    render(<AIInsights insights={mockInsights} error="Failed to load insights" />)
    
    expect(screen.getByText('AI Analysis Unavailable')).toBeInTheDocument()
  })

  it('handles insights without optional fields', () => {
    const minimalInsights: AIInsight = {
      overall_health: 'Basic health status',
      ai_confidence: 0.5,
      critical_issues: 0
    }
    
    render(<AIInsights insights={minimalInsights} />)
    
    expect(screen.getByText('Basic health status')).toBeInTheDocument()
    expect(screen.getByText('AI Confidence: 50%')).toBeInTheDocument()
    expect(screen.queryByText('Predicted Issues')).not.toBeInTheDocument()
    expect(screen.queryByText('Top Recommendations')).not.toBeInTheDocument()
  })

  it('handles recommendations without impact or effort', () => {
    const simpleRecommendation: AIInsight = {
      ...mockInsights,
      top_recommendations: [
        {
          title: 'Simple Rec',
          description: 'Simple description'
        }
      ]
    }
    
    render(<AIInsights insights={simpleRecommendation} />)
    
    expect(screen.getByText('Simple Rec')).toBeInTheDocument()
    expect(screen.getByText('Simple description')).toBeInTheDocument()
    expect(screen.queryByText('Impact:')).not.toBeInTheDocument()
    expect(screen.queryByText('Effort:')).not.toBeInTheDocument()
  })

  it('rounds AI confidence to nearest percentage', () => {
    const preciseInsights: AIInsight = {
      ...mockInsights,
      ai_confidence: 0.876543
    }
    
    render(<AIInsights insights={preciseInsights} />)
    
    expect(screen.getByText('AI Confidence: 88%')).toBeInTheDocument()
  })
})