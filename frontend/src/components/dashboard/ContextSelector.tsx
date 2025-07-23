import { useState, useEffect } from 'react'
import { Check, ChevronsUpDown, Globe } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { config } from '@/config'

interface ContextInfo {
  name: string
  cluster_name: string
  namespace: string
  server: string
  user: string
  current: boolean
}

interface ContextSelectorProps {
  onContextChange?: (context: ContextInfo) => void
}

export function ContextSelector({ onContextChange }: ContextSelectorProps) {
  const [contexts, setContexts] = useState<ContextInfo[]>([])
  const [currentContext, setCurrentContext] = useState<ContextInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [switching, setSwitching] = useState(false)

  // Fetch available contexts
  useEffect(() => {
    fetchContexts()
  }, [])

  const fetchContexts = async () => {
    try {
      const response = await fetch(`${config.api.baseUrl}/api/v1/contexts`)
      if (!response.ok) throw new Error('Failed to fetch contexts')
      const data = await response.json()
      setContexts(data.contexts || [])
      
      // Find current context
      const current = data.contexts?.find((ctx: ContextInfo) => ctx.current)
      if (current) {
        setCurrentContext(current)
      }
    } catch (error) {
      console.error('Error fetching contexts:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleContextSwitch = async (contextName: string) => {
    if (contextName === currentContext?.name) return
    
    setSwitching(true)
    try {
      const response = await fetch(`${config.api.baseUrl}/api/v1/contexts/switch`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ context_name: contextName })
      })
      
      if (!response.ok) throw new Error('Failed to switch context')
      
      const data = await response.json()
      setCurrentContext(data.context)
      
      // Update contexts list to reflect new current
      setContexts(contexts.map(ctx => ({
        ...ctx,
        current: ctx.name === contextName
      })))
      
      // Notify parent component
      if (onContextChange && data.context) {
        onContextChange(data.context)
      }
    } catch (error) {
      console.error('Error switching context:', error)
    } finally {
      setSwitching(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center gap-2 px-3 py-2">
        <Globe className="h-4 w-4 animate-pulse" />
        <span className="text-sm">Loading contexts...</span>
      </div>
    )
  }

  return (
    <Select
      value={currentContext?.name}
      onValueChange={handleContextSwitch}
      disabled={switching}
    >
      <SelectTrigger className="w-[240px] h-9">
        <div className="flex items-center gap-2">
          <Globe className="h-4 w-4" />
          <SelectValue placeholder="Select context">
            {currentContext ? (
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium">{currentContext.name}</span>
                <Badge variant="secondary" className="text-xs px-1.5 py-0">
                  {currentContext.namespace}
                </Badge>
              </div>
            ) : (
              "Select context"
            )}
          </SelectValue>
        </div>
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          <SelectLabel className="text-xs font-semibold uppercase tracking-wider opacity-60">
            Available Contexts
          </SelectLabel>
          {contexts.map((context) => (
            <SelectItem 
              key={context.name} 
              value={context.name}
              className="py-2"
            >
              <div className="flex items-center justify-between w-full">
                <div className="flex flex-col">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{context.name}</span>
                    {context.current && (
                      <Check className="h-3 w-3 text-green-600" />
                    )}
                  </div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span>{context.cluster_name}</span>
                    <span>•</span>
                    <span>{context.namespace}</span>
                  </div>
                </div>
              </div>
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  )
}