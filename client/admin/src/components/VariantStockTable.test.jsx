import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import VariantStockTable from './VariantStockTable'

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    message: {
      success: vi.fn(),
      error: vi.fn(),
    },
  }
})

vi.mock('../lib/useAuth', () => ({
  useAuth: () => ({ authenticated: true, token: 'test-token' }),
}))

const fixtureVariants = [
  {
    sku: 'BAG-001-BRN-M',
    colorName: 'Brown',
    active: true,
    retailPrice: { amountCents: 18900 },
    currency: 'USD',
    stock: { available: 12, reserved: 0, stockStatus: 'IN_STOCK' },
  },
]

function wrapper({ children }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

describe('VariantStockTable available editing', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('click on available activates the input', () => {
    render(<VariantStockTable variants={fixtureVariants} />, { wrapper })

    fireEvent.click(screen.getByRole('button', { name: '12' }))

    expect(screen.getByRole('spinbutton')).toBeInTheDocument()
  })

  it('+10 adds to the current input value', async () => {
    render(<VariantStockTable variants={fixtureVariants} />, { wrapper })

    fireEvent.click(screen.getByRole('button', { name: '12' }))
    fireEvent.click(screen.getByRole('button', { name: '+10' }))

    await waitFor(() => {
      expect(screen.getByRole('spinbutton').value).toBe('22')
    })
  })

  it('Enter calls PUT /admin/inventory/{sku} with correct delta', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ available: 22 }),
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<VariantStockTable variants={fixtureVariants} />, { wrapper })

    fireEvent.click(screen.getByRole('button', { name: '12' }))
    fireEvent.click(screen.getByRole('button', { name: '+10' }))

    // Wait for the value to update before confirming
    await waitFor(() =>
      expect(screen.getByRole('spinbutton').value).toBe('22'),
    )

    fireEvent.keyDown(screen.getByRole('spinbutton'), { key: 'Enter' })

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(1)
      const [url, opts] = fetchMock.mock.calls[0]
      expect(url).toContain('/admin/inventory/BAG-001-BRN-M')
      expect(opts.method).toBe('PUT')
      expect(JSON.parse(opts.body)).toEqual({
        delta: 10,
        reason: 'manual adjustment',
      })
    })
  })

  it('Escape cancels without calling the API', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    render(<VariantStockTable variants={fixtureVariants} />, { wrapper })

    fireEvent.click(screen.getByRole('button', { name: '12' }))
    expect(screen.getByRole('spinbutton')).toBeInTheDocument()

    fireEvent.keyDown(screen.getByRole('spinbutton'), { key: 'Escape' })

    expect(screen.queryByRole('spinbutton')).not.toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('shows success toast after successful stock adjustment', async () => {
    const { message } = await import('antd')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ available: 22 }),
    }))

    render(<VariantStockTable variants={fixtureVariants} />, { wrapper })

    fireEvent.click(screen.getByRole('button', { name: '12' }))
    fireEvent.click(screen.getByRole('button', { name: '+10' }))
    await waitFor(() => expect(screen.getByRole('spinbutton').value).toBe('22'))
    fireEvent.keyDown(screen.getByRole('spinbutton'), { key: 'Enter' })

    await waitFor(() => {
      expect(message.success).toHaveBeenCalledWith('Stock updated for BAG-001-BRN-M', 3)
    })
  })

  it('shows error toast after failed stock adjustment', async () => {
    const { message } = await import('antd')
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
    }))

    render(<VariantStockTable variants={fixtureVariants} />, { wrapper })

    fireEvent.click(screen.getByRole('button', { name: '12' }))
    fireEvent.click(screen.getByRole('button', { name: '+10' }))
    await waitFor(() => expect(screen.getByRole('spinbutton').value).toBe('22'))
    fireEvent.keyDown(screen.getByRole('spinbutton'), { key: 'Enter' })

    await waitFor(() => {
      expect(message.error).toHaveBeenCalledWith('Failed to update stock', 3)
    })
  })
})
