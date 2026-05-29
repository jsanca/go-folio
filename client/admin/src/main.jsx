import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import keycloak from './lib/keycloak'
import App from './App.jsx'
import './index.css'

const queryClient = new QueryClient()

keycloak
  .init({ onLoad: 'check-sso', pkceMethod: 'S256' })
  .then(() => {
    createRoot(document.getElementById('root')).render(
      <StrictMode>
        <QueryClientProvider client={queryClient}>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </QueryClientProvider>
      </StrictMode>,
    )
  })
  .catch((err) => {
    console.error('Keycloak init failed', err)
    document.getElementById('root').textContent = 'Authentication service unavailable.'
  })
