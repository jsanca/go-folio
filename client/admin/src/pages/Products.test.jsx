import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent, act, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Modal } from 'antd'
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
    slug: 'leather-tote-bag',
    variants: [
      { sku: 'BAG-001-BRN-M', active: true },
      { sku: 'BAG-001-BLK-M', active: true },
    ],
  },
  {
    id: 2,
    productCode: 'BELT-001',
    title: 'Classic Belt',
    slug: 'classic-belt',
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

    // BAG-001: all variants active → Active; BELT-001: one inactive → Inactive
    // Scope to tbody to avoid matching the "Active" column header <th>.
    const tbody = document.querySelector('.ant-table-tbody')
    expect(within(tbody).getAllByText('Active')).toHaveLength(1)
    expect(within(tbody).getAllByText('Inactive')).toHaveLength(1)
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

  it('renders Edit and Delete buttons for each row', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => fixtureProducts,
    }))

    render(<Products />, { wrapper })

    await screen.findByText('BAG-001')

    // One Edit + Delete pair per row
    expect(screen.getAllByRole('button', { name: 'Edit' })).toHaveLength(2)
    expect(screen.getAllByRole('button', { name: 'Delete' })).toHaveLength(2)
  })
})

describe('Products mutations', () => {
  function makeFetch({ getResp = fixtureProducts, postResp = { id: 3, productCode: 'NEW-001', title: 'New' }, patchResp = fixtureProducts[0], deleteStatus = 204 } = {}) {
    return vi.fn().mockImplementation((url, options) => {
      const method = options?.method ?? 'GET'
      if (method === 'GET') return Promise.resolve({ ok: true, json: async () => getResp })
      if (method === 'POST') return Promise.resolve({ ok: true, json: async () => postResp })
      if (method === 'PATCH') return Promise.resolve({ ok: true, json: async () => patchResp })
      if (method === 'DELETE') return Promise.resolve({ ok: deleteStatus < 400, status: deleteStatus, json: async () => ({}) })
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
  }

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    Modal.destroyAll()
  })

  it('opens Create modal when Create Product button is clicked', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(<Products />, { wrapper })

    // Wait for table to load
    await screen.findByText('BAG-001')

    fireEvent.click(screen.getByRole('button', { name: 'Create Product' }))

    expect(await screen.findByText('Create Product', { selector: '.ant-modal-title' })).toBeInTheDocument()
  })

  it('POST /admin/products is called with correct payload on create submit', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(<Products />, { wrapper })

    await screen.findByText('BAG-001')

    // Open create modal
    fireEvent.click(screen.getByRole('button', { name: 'Create Product' }))
    await screen.findByText('Create Product', { selector: '.ant-modal-title' })

    // Fill required fields
    fireEvent.change(screen.getByLabelText('Product Code'), {
      target: { value: 'NEW-001' },
    })
    fireEvent.change(screen.getByLabelText('Title'), {
      target: { value: 'New Product' },
    })
    // Slug auto-fills from title
    await waitFor(() =>
      expect(screen.getByLabelText('Slug').value).toBe('new-product'),
    )

    // Click Create (modal OK button)
    fireEvent.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => {
      const postCall = fetchMock.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeDefined()
      const body = JSON.parse(postCall[1].body)
      expect(body.productCode).toBe('NEW-001')
      expect(body.title).toBe('New Product')
      expect(body.slug).toBe('new-product')
    })
  })

  it('PATCH /admin/products/:id is called with correct payload on edit submit', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(<Products />, { wrapper })

    await screen.findByText('BAG-001')

    // Click Edit on first row
    fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[0])
    await screen.findByText('Edit Product', { selector: '.ant-modal-title' })

    // Update the title
    const titleInput = screen.getByLabelText('Title')
    fireEvent.change(titleInput, { target: { value: 'Updated Bag' } })

    // Click Save
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      const patchCall = fetchMock.mock.calls.find(([, opts]) => opts?.method === 'PATCH')
      expect(patchCall).toBeDefined()
      expect(patchCall[0]).toContain('/admin/products/1')
      const body = JSON.parse(patchCall[1].body)
      expect(body.title).toBe('Updated Bag')
    })
  })

  it('shows Delete confirmation modal before deleting', async () => {
    vi.stubGlobal('fetch', makeFetch())
    render(<Products />, { wrapper })

    await screen.findByText('BAG-001')

    fireEvent.click(within(document.querySelector('.ant-table-tbody')).getAllByRole('button', { name: 'Delete' })[0])

    // Ant Design Modal.confirm title appears in the document
    expect(await screen.findByText('Delete product?', { selector: '.ant-modal-title' })).toBeInTheDocument()
    expect(screen.getByText(/"Leather Tote Bag" will be permanently removed\./)).toBeInTheDocument()
  })

  it('calls DELETE /admin/products/:id when confirm is clicked', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(<Products />, { wrapper })

    await screen.findByText('BAG-001')

    const prevConfirmCount = document.querySelectorAll('.ant-modal-confirm').length
    fireEvent.click(within(document.querySelector('.ant-table-tbody')).getAllByRole('button', { name: 'Delete' })[0])

    // Wait for a NEW confirm dialog to appear
    await waitFor(() => {
      expect(document.querySelectorAll('.ant-modal-confirm').length).toBeGreaterThan(prevConfirmCount)
    })

    // Click the Delete button inside the newest confirm dialog
    const confirms = document.querySelectorAll('.ant-modal-confirm')
    fireEvent.click(within(confirms[confirms.length - 1]).getByRole('button', { name: 'Delete' }))

    await waitFor(() => {
      const deleteCall = fetchMock.mock.calls.find(([, opts]) => opts?.method === 'DELETE')
      expect(deleteCall).toBeDefined()
      expect(deleteCall[0]).toContain('/admin/products/1')
    })
  })

  it('re-fetches the product list after a successful mutation', async () => {
    const fetchMock = makeFetch()
    vi.stubGlobal('fetch', fetchMock)
    render(<Products />, { wrapper })

    await screen.findByText('BAG-001')

    const initialGetCount = fetchMock.mock.calls.filter(
      ([, opts]) => !opts?.method || opts.method === 'GET',
    ).length

    // Trigger a delete
    const prevConfirmCount = document.querySelectorAll('.ant-modal-confirm').length
    fireEvent.click(within(document.querySelector('.ant-table-tbody')).getAllByRole('button', { name: 'Delete' })[0])

    // Wait for a NEW confirm dialog to appear
    await waitFor(() => {
      expect(document.querySelectorAll('.ant-modal-confirm').length).toBeGreaterThan(prevConfirmCount)
    })

    // Click the Delete button inside the newest confirm dialog
    const confirms = document.querySelectorAll('.ant-modal-confirm')
    fireEvent.click(within(confirms[confirms.length - 1]).getByRole('button', { name: 'Delete' }))

    // Wait for the DELETE to complete
    await waitFor(() => {
      expect(fetchMock.mock.calls.find(([, opts]) => opts?.method === 'DELETE')).toBeDefined()
    })

    // After successful delete the query is invalidated → GET is called again
    await waitFor(() => {
      const getCount = fetchMock.mock.calls.filter(
        ([, opts]) => !opts?.method || opts.method === 'GET',
      ).length
      expect(getCount).toBeGreaterThan(initialGetCount)
    })
  })
})

describe('Products expandable rows', () => {
  const productWithVariants = {
    id: 10,
    productCode: 'WALLET-001',
    title: 'Slim Wallet',
    slug: 'slim-wallet',
    variants: [
      {
        sku: 'WALLET-001-BRN-OS',
        colorName: 'Brown',
        active: true,
        retailPrice: { amountCents: 4999, currency: 'USD' },
        stock: { available: 12, reserved: 0, stockStatus: 'IN_STOCK' },
      },
      {
        sku: 'WALLET-001-BLK-OS',
        colorName: 'Black',
        active: true,
        retailPrice: { amountCents: 4999, currency: 'USD' },
        stock: { available: 3, reserved: 1, stockStatus: 'LOW_STOCK' },
      },
    ],
  }

  const productNoVariants = {
    id: 11,
    productCode: 'EMPTY-001',
    title: 'No Variants',
    slug: 'no-variants',
    variants: [],
  }

  function makeExpandFetch(products) {
    return vi.fn().mockResolvedValue({ ok: true, json: async () => products })
  }

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('expanded row renders variant SKU and stock status', async () => {
    vi.stubGlobal('fetch', makeExpandFetch([productWithVariants]))

    render(<Products />, { wrapper })
    await screen.findByText('WALLET-001')

    // Click the expand button for the row
    const expandBtn = document.querySelector('button.ant-table-row-expand-icon')
    fireEvent.click(expandBtn)

    await screen.findByText('WALLET-001-BRN-OS')
    expect(screen.getByText('WALLET-001-BLK-OS')).toBeInTheDocument()
    expect(screen.getByText('In Stock')).toBeInTheDocument()
    expect(screen.getByText('Low Stock')).toBeInTheDocument()
  })

  it('row with 0 variants is not expandable', async () => {
    vi.stubGlobal('fetch', makeExpandFetch([productWithVariants, productNoVariants]))

    render(<Products />, { wrapper })
    await screen.findByText('WALLET-001')

    // The row with variants has ant-table-row-expand-icon-collapsed; the empty
    // row still renders a button but with ant-table-row-expand-icon-spanned
    // (Ant Design's "not expandable" marker). Only the collapsed variant counts.
    const clickableExpanders = document.querySelectorAll(
      'button.ant-table-row-expand-icon-collapsed',
    )
    expect(clickableExpanders).toHaveLength(1)
  })
})
