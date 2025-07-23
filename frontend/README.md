# KubePulse Frontend Dashboard

React + TypeScript dashboard for KubePulse Kubernetes monitoring with AI-powered insights.

## Overview

The KubePulse frontend is a modern React application that provides:

- **Real-time Cluster Health Visualization** - Live monitoring dashboard
- **AI-Powered Insights** - Interactive AI assistant and analysis results  
- **WebSocket Integration** - Live updates without page refresh
- **Responsive Design** - Works on desktop, tablet, and mobile
- **TypeScript** - Full type safety for better development experience

## Features

### 🎛️ Dashboard Components

- **Health Status Overview** - Cluster-wide health at a glance
- **Pod Monitoring** - Real-time pod status and metrics
- **Node Health** - Node resource usage and availability
- **Service Status** - Service endpoint health and connectivity

### 🤖 AI Integration

- **AI Assistant Chat** - Natural language queries about your cluster
- **Predictive Analytics** - Failure forecasting and trend analysis
- **Auto-remediation UI** - Review and execute AI-generated fixes
- **Smart Alerts** - Intelligent alert correlation and noise reduction

### ⚡ Real-time Features

- **WebSocket Updates** - Live cluster data streaming
- **Auto-refresh** - Configurable data refresh intervals
- **Connection Management** - Automatic reconnection handling
- **Performance Optimized** - Efficient rendering of large datasets

## Development

### Prerequisites

- Node.js 18+ and npm/yarn
- KubePulse backend running on `http://localhost:8080` (configurable)

### Setup

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production  
npm run build

# Preview production build
npm run preview

# Run type checking
npm run type-check

# Run linting
npm run lint
```

### Environment Variables

Create a `.env.local` file in the frontend directory:

```env
# Backend API URL (default: http://localhost:8080)
VITE_API_BASE_URL=http://localhost:8080

# WebSocket URL (default: ws://localhost:8080)
VITE_WS_URL=ws://localhost:8080

# Enable development features
VITE_DEV_MODE=true
```

### Project Structure

```
frontend/
├── public/                 # Static assets
├── src/
│   ├── components/         # React components
│   │   ├── dashboard/      # Dashboard-specific components
│   │   │   ├── HealthOverview.tsx
│   │   │   ├── ClusterMetrics.tsx
│   │   │   └── AIInsights.tsx
│   │   ├── layout/         # Layout components
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   └── Layout.tsx
│   │   └── ui/             # Reusable UI components
│   │       ├── Card.tsx
│   │       ├── Button.tsx
│   │       └── Loading.tsx
│   ├── hooks/              # Custom React hooks
│   │   ├── useWebSocket.ts # WebSocket connection management
│   │   ├── useAIInsights.ts # AI data fetching
│   │   └── useClusterHealth.ts # Health data management
│   ├── lib/                # Utilities and configurations
│   │   ├── api.ts          # API client setup
│   │   ├── types.ts        # TypeScript type definitions
│   │   └── utils.ts        # Helper functions
│   ├── pages/              # Page components
│   │   ├── Dashboard.tsx   # Main dashboard
│   │   ├── AIChat.tsx      # AI assistant interface
│   │   └── Settings.tsx    # Configuration page
│   ├── App.tsx             # Main application component
│   └── main.tsx            # Application entry point
├── package.json            # Dependencies and scripts
├── tsconfig.json           # TypeScript configuration
├── vite.config.ts          # Vite build configuration
└── tailwind.config.js      # Tailwind CSS configuration
```

## Key Hooks and Components

### useWebSocket Hook

```typescript
import { useWebSocket } from './hooks/useWebSocket';

function Dashboard() {
  const { data, connectionState, error } = useWebSocket(
    'ws://localhost:8080/ws'
  );
  
  // Handle real-time updates
}
```

### useAIInsights Hook

```typescript
import { useAIInsights } from './hooks/useAIInsights';

function AIPanel() {
  const { 
    insights, 
    askQuestion, 
    isLoading 
  } = useAIInsights();
  
  // AI interaction logic
}
```

### Health Status Components

```typescript
import { HealthOverview } from './components/dashboard/HealthOverview';
import { ClusterMetrics } from './components/dashboard/ClusterMetrics';

function Dashboard() {
  return (
    <div className="dashboard">
      <HealthOverview />
      <ClusterMetrics />
    </div>
  );
}
```

## API Integration

The frontend integrates with the KubePulse backend API:

### REST API Endpoints

```typescript
// Health data
const healthData = await fetch('/api/v1/health/cluster');

// AI assistant query
const aiResponse = await fetch('/api/v1/ai/assistant/query', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ query: 'Why is my pod failing?' })
});

// Remediation actions
const remediation = await fetch('/api/v1/ai/remediation/pod-check/suggestions');
```

### WebSocket Integration

```typescript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const healthUpdate = JSON.parse(event.data);
  // Update dashboard with real-time data
};
```

## Styling

The application uses:

- **Tailwind CSS** - Utility-first CSS framework
- **Headless UI** - Unstyled, accessible UI components
- **Heroicons** - Beautiful SVG icons
- **Custom CSS Variables** - Theme customization support

## Performance

- **Code Splitting** - Automatic route-based splitting
- **Lazy Loading** - Components loaded on demand
- **Memoization** - Optimized re-rendering
- **Virtual Scrolling** - Efficient large list rendering

## Testing

```bash
# Run unit tests
npm run test

# Run e2e tests
npm run test:e2e

# Run tests in watch mode
npm run test:watch

# Generate coverage report
npm run test:coverage
```

## Deployment

### Production Build

```bash
# Build optimized production bundle
npm run build

# The built files will be in the dist/ directory
# Copy these to your web server or CDN
```

### Docker Deployment

The frontend is included in the main KubePulse Docker image and served by the Go backend for a single binary deployment.

## Contributing

1. Follow the existing code style and structure
2. Use TypeScript for all new code
3. Add appropriate error handling and loading states
4. Write unit tests for complex components
5. Update this README when adding new features

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## License

Same as KubePulse main project - Apache License 2.0.