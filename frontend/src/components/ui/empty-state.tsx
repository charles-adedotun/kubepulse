import { RefreshCw, AlertCircle, Server, Database, WifiOff } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

interface EmptyStateCardProps {
  state: 'loading' | 'no_contexts' | 'disconnected' | 'invalid_context' | 'no_data'
  title?: string
  message?: string
  error?: string
  suggestions?: string[]
  canRetry?: boolean
  onRetry?: () => void
  isRetrying?: boolean
  className?: string
}

const stateConfig = {
  loading: {
    icon: RefreshCw,
    title: 'Loading...',
    message: 'Checking connection status',
    variant: 'default' as const,
    iconClassName: 'animate-spin'
  },
  no_contexts: {
    icon: Database,
    title: 'No Kubernetes Contexts',
    message: 'No Kubernetes contexts found. Configure kubectl to get started.',
    variant: 'destructive' as const,
    iconClassName: ''
  },
  disconnected: {
    icon: WifiOff,
    title: 'Connection Failed',
    message: 'Unable to connect to the Kubernetes cluster.',
    variant: 'destructive' as const,
    iconClassName: ''
  },
  invalid_context: {
    icon: AlertCircle,
    title: 'Invalid Context',
    message: 'The current Kubernetes context is not valid.',
    variant: 'secondary' as const,
    iconClassName: ''
  },
  no_data: {
    icon: Server,
    title: 'No Data Available',
    message: 'No monitoring data available at this time.',
    variant: 'secondary' as const,
    iconClassName: ''
  }
}

export function EmptyStateCard({
  state,
  title,
  message,
  error,
  suggestions = [],
  canRetry = false,
  onRetry,
  isRetrying = false,
  className = ''
}: EmptyStateCardProps) {
  const config = stateConfig[state]
  const Icon = config.icon
  
  const displayTitle = title || config.title
  const displayMessage = message || config.message

  return (
    <Card className={`${className} border-dashed`}>
      <CardHeader className="text-center pb-4">
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-gray-800">
          <Icon className={`h-6 w-6 text-gray-600 dark:text-gray-400 ${config.iconClassName}`} />
        </div>
        <CardTitle className="text-lg">{displayTitle}</CardTitle>
        <CardDescription className="text-sm text-gray-600 dark:text-gray-400">
          {displayMessage}
        </CardDescription>
      </CardHeader>
      
      <CardContent className="space-y-4">
        {error && (
          <div className="rounded-md bg-red-50 dark:bg-red-900/20 p-3 border border-red-200 dark:border-red-800">
            <div className="flex items-start">
              <AlertCircle className="h-4 w-4 text-red-500 mt-0.5 mr-2 flex-shrink-0" />
              <div className="text-sm text-red-700 dark:text-red-300">
                <strong>Error:</strong> {error}
              </div>
            </div>
          </div>
        )}

        {suggestions.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
              Suggested actions:
            </h4>
            <ul className="space-y-1">
              {suggestions.map((suggestion, index) => (
                <li key={index} className="flex items-start">
                  <span className="inline-block w-1.5 h-1.5 bg-gray-400 rounded-full mr-2 mt-2 flex-shrink-0" />
                  <span className="text-sm text-gray-600 dark:text-gray-400">{suggestion}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {canRetry && onRetry && (
          <div className="flex justify-center pt-2">
            <Button
              onClick={onRetry}
              disabled={isRetrying}
              variant="outline"
              size="sm"
              className="flex items-center gap-2"
            >
              <RefreshCw className={`h-4 w-4 ${isRetrying ? 'animate-spin' : ''}`} />
              {isRetrying ? 'Checking...' : 'Retry Connection'}
            </Button>
          </div>
        )}

        {state === 'no_contexts' && (
          <div className="mt-4 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md">
            <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-2">
              Getting Started
            </h4>
            <div className="space-y-2 text-sm text-blue-700 dark:text-blue-300">
              <p>To set up Kubernetes access:</p>
              <ol className="list-decimal list-inside space-y-1 ml-2">
                <li>Configure your kubeconfig file</li>
                <li>Add a cluster connection</li>
                <li>Set the current context</li>
              </ol>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// Convenience components for specific states
export function LoadingStateCard({ className = '', ...props }: Omit<EmptyStateCardProps, 'state'>) {
  return <EmptyStateCard state="loading" className={className} {...props} />
}

export function NoContextsCard({ className = '', ...props }: Omit<EmptyStateCardProps, 'state'>) {
  return <EmptyStateCard state="no_contexts" className={className} {...props} />
}

export function DisconnectedCard({ className = '', ...props }: Omit<EmptyStateCardProps, 'state'>) {
  return <EmptyStateCard state="disconnected" className={className} {...props} />
}

export function InvalidContextCard({ className = '', ...props }: Omit<EmptyStateCardProps, 'state'>) {
  return <EmptyStateCard state="invalid_context" className={className} {...props} />
}

export function NoDataCard({ className = '', ...props }: Omit<EmptyStateCardProps, 'state'>) {
  return <EmptyStateCard state="no_data" className={className} {...props} />
}