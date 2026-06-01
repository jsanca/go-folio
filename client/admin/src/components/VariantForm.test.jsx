import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import VariantForm from './VariantForm'

vi.mock('../lib/useAuth', () => ({
  useAuth: () => ({ authenticated: true, token: 'test-token' }),
}))

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    message: { success: vi.fn(), error: vi.fn() },
  }
})

function wrapper({ children }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

function renderForm(props = {}) {
  return render(
    <VariantForm productId={42} open={true} onClose={vi.fn()} {...props} />,
    { wrapper },
  )
}

describe('VariantForm', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders all fields', () => {
    renderForm()

    expect(screen.getByLabelText('SKU')).toBeInTheDocument()
    expect(screen.getByLabelText('Color Name')).toBeInTheDocument()
    expect(screen.getByLabelText('Color Slug')).toBeInTheDocument()
    expect(screen.getByLabelText('Primary Color')).toBeInTheDocument()
    expect(screen.getByLabelText('Retail Price (cents)')).toBeInTheDocument()
    expect(screen.getByLabelText('Currency')).toBeInTheDocument()
    // Switch is not a standard input — verify by label text in DOM
    expect(screen.getByText('Active')).toBeInTheDocument()
  })

  it('auto-generates colorSlug from colorName', async () => {
    renderForm()

    fireEvent.change(screen.getByLabelText('Color Name'), {
      target: { value: 'Rojo Oscuro' },
    })

    await waitFor(() => {
      expect(screen.getByLabelText('Color Slug').value).toBe('rojo-oscuro')
    })
  })

  it('calls POST /admin/products/{id}/variants with correct payload on submit', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ sku: 'BAG-RED', currency: 'CRC' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    renderForm({ productId: 7 })

    fireEvent.change(screen.getByLabelText('SKU'), {
      target: { value: 'BAG-RED' },
    })
    // retailPriceCents — InputNumber renders a spinbutton
    fireEvent.change(screen.getByRole('spinbutton'), {
      target: { value: '189000' },
    })

    fireEvent.click(screen.getByRole('button', { name: 'Add Variant' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(1)
      const [url, opts] = fetchMock.mock.calls[0]
      expect(url).toContain('/admin/products/7/variants')
      expect(opts.method).toBe('POST')
      const body = JSON.parse(opts.body)
      expect(body.sku).toBe('BAG-RED')
      expect(body.currency).toBe('CRC')
    })
  })

  it('invalidates admin-products query after successful submit', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ sku: 'BAG-RED', currency: 'CRC' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const invalidateSpy = vi.spyOn(qc, 'invalidateQueries')

    render(
      <QueryClientProvider client={qc}>
        <VariantForm productId={7} open={true} onClose={vi.fn()} />
      </QueryClientProvider>,
    )

    fireEvent.change(screen.getByLabelText('SKU'), {
      target: { value: 'BAG-RED' },
    })
    fireEvent.change(screen.getByRole('spinbutton'), {
      target: { value: '189000' },
    })

    fireEvent.click(screen.getByRole('button', { name: 'Add Variant' }))

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: ['admin-products'],
      })
    })
  })
})
