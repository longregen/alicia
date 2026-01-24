import React from 'react'
import ReactDOM from 'react-dom/client'
import { initOtel } from './lib/otel'
import App from './App'
import './index.css'
import { ErrorBoundary } from './components/ErrorBoundary'

// Initialize OpenTelemetry before React renders
// This ensures all fetch requests are instrumented from the start
initOtel('alicia-web')

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>,
)
