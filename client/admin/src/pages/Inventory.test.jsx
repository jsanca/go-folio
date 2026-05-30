import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import Inventory from './Inventory'

vi.mock('../lib/useAuth', () => ({
  useAuth: () => ({ authenticated: true, token: 'test-token' }),
}))

const fixtureInventory = [
  { sku: 'BAG-001-BRN', available: 20, reserved: 2,  status: 'IN_STOCK'     },
  { sku: 'BAG-001-BLK', available:  3, reserved: 1,  status: 'LOW_STOCK'    },
  { sku: 'BAG-001-GRN', available:  0, reserved: 0,  status: 'OUT_OF_STOCK' },
]

function wrapper({ children }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

beforeEach(() => {
  vi.restoreAllMocks()
})

// ── read-only rendering ───────────────────────────────────────────────────────

describe('Inventory', () => {
  it('renders table with fetched inventory data', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => fixtureInventory,
    }))

    render(<Inventory />, { wrapper })

    await screen.findByText('BAG-001-BRN')
    expect(screen.getByText('BAG-001-BLK')).toBeInTheDocument()
    expect(screen.getByText('BAG-001-GRN')).toBeInTheDocument()
    expect(screen.getByText('IN STOCK')).toBeInTheDocument()
    expect(screen.getByText('LOW STOCK')).toBeInTheDocument()
    expect(screen.getByText('OUT OF STOCK')).toBeInTheDocument()
  })

  it('shows a spinner while loading', async () => {
    let resolve
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(
      new Promise((r) => { resolve = r }),
    ))

    render(<Inventory />, { wrapper })

    await waitFor(() =>
      expect(document.querySelector('.ant-spin')).toBeInTheDocument(),
    )

    resolve({ ok: true, json: async () => [] })
  })

  it('shows an error alert when fetch fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
    }))

    render(<Inventory />, { wrapper })

    await screen.findByText('Failed to load inventory')
    expect(screen.getByText(/GET \/admin\/inventory → 503/)).toBeInTheDocument()
  })
})

// ── adjustments ───────────────────────────────────────────────────────────────

describe('Inventory adjustments', () => {
  function makeFetch({
    getResp    = fixtureInventory,
    putStatus  = 200,
    putResp    = { sku: 'BAG-001-BRN', available: 18, status: 'IN_STOCK' },
  } = {}) {
    return vi.fn().mockImplementation((url, options) => {
      const method = options?.method ?? 'GET'
      if (method === 'GET')
        return Promise.resolve({ ok: true, json: async () => getResp })
      if (method === 'PUT')
        return Promise.resolve({
          ok: putStatus < 400,
          status: putStatus,
          json: async () => putResp,
        })
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
  }

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    // nothing to clean up — AdjustStockForm is state-controlled, not Modal.confirm
  })

  it('opens Adjust modal when Adjust button is clicked', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(<Inventory />, { wrapper })

    await screen.findByText('BAG-001-BRN')

    fireEvent.click(
      within(document.querySelector('.ant-table-tbody'))
        .getAllByRole('button', { name: 'Adjust' })[0],
    )

    expect(
      await screen.findByText(/Adjust Stock/, { selector: '.ant-modal-title' }),
    ).toBeInTheDocument()
  })

  it('calls PUT /admin/inventory/{sku} with correct payload', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(<Inventory />, { wrapper })

    await screen.findByText('BAG-001-BRN')

    // Open modal for the first row (BAG-001-BRN)
    fireEvent.click(
      within(document.querySelector('.ant-table-tbody'))
        .getAllByRole('button', { name: 'Adjust' })[0],
    )
    await screen.findByText(/Adjust Stock/, { selector: '.ant-modal-title' })

    fireEvent.change(screen.getByLabelText('Delta'), { target: { value: '-2' } })
    fireEvent.change(screen.getByLabelText('Reason'), { target: { value: 'sale' } })

    // The modal OK button
    fireEvent.click(
      within(document.querySelector('.ant-modal-footer'))
        .getByRole('button', { name: 'Adjust' }),
    )

    await waitFor(() => {
      const putCall = fetchMock.mock.calls.find(([, opts]) => opts?.method === 'PUT')
      expect(putCall).toBeDefined()
      expect(putCall[0]).toContain('/admin/inventory/BAG-001-BRN')
      const body = JSON.parse(putCall[1].body)
      expect(body.delta).toBe(-2)
      expect(body.reason).toBe('sale')
    })
  })

  it('re-fetches the inventory list after a successful adjustment', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(<Inventory />, { wrapper })

    await screen.findByText('BAG-001-BRN')

    const initialGetCount = fetchMock.mock.calls.filter(
      ([, opts]) => !opts?.method || opts.method === 'GET',
    ).length

    fireEvent.click(
      within(document.querySelector('.ant-table-tbody'))
        .getAllByRole('button', { name: 'Adjust' })[0],
    )
    await screen.findByText(/Adjust Stock/, { selector: '.ant-modal-title' })

    fireEvent.change(screen.getByLabelText('Delta'), { target: { value: '5' } })

    fireEvent.click(
      within(document.querySelector('.ant-modal-footer'))
        .getByRole('button', { name: 'Adjust' }),
    )

    // Wait for the PUT to complete
    await waitFor(() => {
      expect(
        fetchMock.mock.calls.find(([, opts]) => opts?.method === 'PUT'),
      ).toBeDefined()
    })

    // After success the query is invalidated → GET fires again
    await waitFor(() => {
      const getCount = fetchMock.mock.calls.filter(
        ([, opts]) => !opts?.method || opts.method === 'GET',
      ).length
      expect(getCount).toBeGreaterThan(initialGetCount)
    })
  })

  it('shows inline error "Insufficient stock" on 422', async () => {
    const fetchMock = makeFetch({ putStatus: 422 })
    vi.stubGlobal('fetch', fetchMock)
    render(<Inventory />, { wrapper })

    await screen.findByText('BAG-001-BRN')

    fireEvent.click(
      within(document.querySelector('.ant-table-tbody'))
        .getAllByRole('button', { name: 'Adjust' })[0],
    )
    await screen.findByText(/Adjust Stock/, { selector: '.ant-modal-title' })

    fireEvent.change(screen.getByLabelText('Delta'), { target: { value: '-999' } })

    fireEvent.click(
      within(document.querySelector('.ant-modal-footer'))
        .getByRole('button', { name: 'Adjust' }),
    )

    await screen.findByText('Insufficient stock')
  })
})
