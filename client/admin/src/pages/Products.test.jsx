import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Products from './Products'

// Mock useAuth so enabled: authenticated && !!token is always satisfied.
// Products.test.jsx tests rendering logic, not auth state transitions.
vi.mock('../lib/useAuth', () => ({
  useAuth: () => ({ authenticated: true, token: 'test-token' }),
}))

const fixtureProducts = [
  {
    id: 1,
    productCode: 'BAG-001',
    title: 'Leather Tote Bag',
    variants: [
      { sku: 'BAG-001-BRN-M', active: true },
      { sku: 'BAG-001-BLK-M', active: true },
    ],
  },
  {
    id: 2,
    productCode: 'BELT-001',
    title: 'Classic Belt',
    variants: [{ sku: 'BELT-001-BRN-M', active: false }],
  },
]

function wrapper({ children }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('Products', () => {
  it('renders table with fetched products', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => fixtureProducts,
    }))

    render(<Products />, { wrapper })

    await screen.findByText('BAG-001')
    expect(screen.getByText('Leather Tote Bag')).toBeInTheDocument()
    expect(screen.getByText('BELT-001')).toBeInTheDocument()
    expect(screen.getByText('Classic Belt')).toBeInTheDocument()

    // BAG-001: all variants active → Yes; BELT-001: one inactive → No
    expect(screen.getAllByText('Yes')).toHaveLength(1)
    expect(screen.getAllByText('No')).toHaveLength(1)
  })

  it('shows a spinner while loading', async () => {
    let resolve
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise((r) => { resolve = r })))

    render(<Products />, { wrapper })

    await waitFor(() => expect(document.querySelector('.ant-spin')).toBeInTheDocument())

    resolve({ ok: true, json: async () => [] })
  })

  it('shows an error alert when fetch fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
    }))

    render(<Products />, { wrapper })

    await screen.findByText('Failed to load products')
    expect(screen.getByText(/GET \/admin\/products → 503/)).toBeInTheDocument()
  })

  it('sends Authorization header with keycloak token', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [],
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<Products />, { wrapper })

    await waitFor(() => expect(fetchMock).toHaveBeenCalled())

    const [, options] = fetchMock.mock.calls[0]
    expect(options.headers.Authorization).toBe('Bearer test-token')
  })
})
