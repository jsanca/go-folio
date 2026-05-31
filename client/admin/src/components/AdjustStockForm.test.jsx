import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import AdjustStockForm from './AdjustStockForm'

vi.mock('../lib/useAuth', () => ({
  useAuth: () => ({ authenticated: true, token: 'test-token' }),
}))

const mockMessageSuccess = vi.hoisted(() => vi.fn())

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal()
  return { ...actual, message: { ...actual.message, success: mockMessageSuccess } }
})

function makeQC() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
}

function wrapper({ children }) {
  return <QueryClientProvider client={makeQC()}>{children}</QueryClientProvider>
}

beforeEach(() => {
  vi.restoreAllMocks()
  mockMessageSuccess.mockClear()
})

describe('AdjustStockForm', () => {
  it('shows the SKU in the modal title', () => {
    render(<AdjustStockForm sku="BAG-001-BRN" open={true} onClose={vi.fn()} />, { wrapper })

    expect(
      screen.getByText('Adjust Stock — BAG-001-BRN', { selector: '.ant-modal-title *' }),
    ).toBeInTheDocument()
  })

  it('Cancel button calls onClose', () => {
    const onClose = vi.fn()
    render(<AdjustStockForm sku="BAG-001-BRN" open={true} onClose={onClose} />, { wrapper })

    fireEvent.click(
      within(document.querySelector('.ant-modal-footer'))
        .getByRole('button', { name: 'Cancel' }),
    )

    expect(onClose).toHaveBeenCalledOnce()
  })

  it('calls PUT /admin/inventory/{sku} with correct payload on submit', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ sku: 'BAG-001-BRN', available: 18, status: 'IN_STOCK' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<AdjustStockForm sku="BAG-001-BRN" open={true} onClose={vi.fn()} />, { wrapper })

    fireEvent.change(screen.getByLabelText('Delta'), { target: { value: '-2' } })
    fireEvent.change(screen.getByLabelText('Reason'), { target: { value: 'sale' } })

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

  it('shows "SKU not found" inline error on 404 response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: async () => ({}),
    }))

    render(<AdjustStockForm sku="GHOST-SKU" open={true} onClose={vi.fn()} />, { wrapper })

    fireEvent.change(screen.getByLabelText('Delta'), { target: { value: '5' } })

    fireEvent.click(
      within(document.querySelector('.ant-modal-footer'))
        .getByRole('button', { name: 'Adjust' }),
    )

    await screen.findByText('SKU not found')
  })

  it('shows a success toast with the SKU after a successful adjustment', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ sku: 'BAG-001-BRN', available: 22, status: 'IN_STOCK' }),
    }))

    render(<AdjustStockForm sku="BAG-001-BRN" open={true} onClose={vi.fn()} />, { wrapper })

    fireEvent.change(screen.getByLabelText('Delta'), { target: { value: '2' } })

    fireEvent.click(
      within(document.querySelector('.ant-modal-footer'))
        .getByRole('button', { name: 'Adjust' }),
    )

    await waitFor(() => {
      expect(mockMessageSuccess).toHaveBeenCalledWith('Stock updated for BAG-001-BRN', 3)
    })
  })

  it('form is empty when modal is reopened after cancel (destroyOnHidden)', async () => {
    const onClose = vi.fn()
    const qc = makeQC()

    const { rerender } = render(
      <QueryClientProvider client={qc}>
        <AdjustStockForm sku="BAG-001-BRN" open={true} onClose={onClose} />
      </QueryClientProvider>,
    )

    // Fill both fields
    fireEvent.change(screen.getByLabelText('Delta'), { target: { value: '42' } })
    fireEvent.change(screen.getByLabelText('Reason'), { target: { value: 'restock' } })

    // Cancel resets the form and signals the parent to close
    fireEvent.click(
      within(document.querySelector('.ant-modal-footer'))
        .getByRole('button', { name: 'Cancel' }),
    )

    // Simulate the parent cycling open false → true (as Inventory.jsx does via adjustSKU state)
    rerender(
      <QueryClientProvider client={qc}>
        <AdjustStockForm sku="BAG-001-BRN" open={false} onClose={onClose} />
      </QueryClientProvider>,
    )
    rerender(
      <QueryClientProvider client={qc}>
        <AdjustStockForm sku="BAG-001-BRN" open={true} onClose={onClose} />
      </QueryClientProvider>,
    )

    await waitFor(() => {
      expect(screen.getByLabelText('Delta').value).toBe('')
      expect(screen.getByLabelText('Reason').value).toBe('')
    })
  })
})
