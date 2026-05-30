import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { Form } from 'antd'
import ProductForm, { slugify } from './ProductForm'

// Renders ProductForm inside a component that owns the Form instance.
// Returns a getter for the form instance so tests can inspect field values.
function renderForm(props = {}) {
  let formInstance
  function Wrapper() {
    const [form] = Form.useForm()
    formInstance = form
    return <ProductForm form={form} {...props} />
  }
  render(<Wrapper />)
  return { getForm: () => formInstance }
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('slugify', () => {
  it('lowercases and replaces spaces with hyphens', () => {
    expect(slugify('Hello World')).toBe('hello-world')
  })

  it('strips leading and trailing hyphens', () => {
    expect(slugify('  hello  ')).toBe('hello')
  })

  it('collapses consecutive non-alphanumeric chars', () => {
    expect(slugify('Leather & Canvas Bag!')).toBe('leather-canvas-bag')
  })

  it('handles empty string', () => {
    expect(slugify('')).toBe('')
  })
})

describe('ProductForm', () => {
  it('renders all fields', () => {
    renderForm()

    expect(screen.getByLabelText('Product Code')).toBeInTheDocument()
    expect(screen.getByLabelText('Title')).toBeInTheDocument()
    expect(screen.getByLabelText('Slug')).toBeInTheDocument()
    expect(screen.getByLabelText('Short Description')).toBeInTheDocument()
    // Department and Category render as comboboxes; verify labels are present
    expect(screen.getByText('Department')).toBeInTheDocument()
    expect(screen.getByText('Category')).toBeInTheDocument()
    // Active switch
    expect(screen.getByRole('switch')).toBeInTheDocument()
  })

  it('auto-generates slug from title when isEdit is false', async () => {
    renderForm({ isEdit: false })

    const titleInput = screen.getByLabelText('Title')
    fireEvent.change(titleInput, { target: { value: 'Leather Tote Bag' } })

    await waitFor(() => {
      expect(screen.getByLabelText('Slug').value).toBe('leather-tote-bag')
    })
  })

  it('does not auto-generate slug when isEdit is true', async () => {
    const { getForm } = renderForm({ isEdit: true })

    // Pre-fill slug as if loaded from an existing product
    act(() => {
      getForm().setFieldsValue({ slug: 'existing-slug' })
    })

    const titleInput = screen.getByLabelText('Title')
    fireEvent.change(titleInput, { target: { value: 'New Title' } })

    await waitFor(() => {
      // slug must remain unchanged
      expect(screen.getByLabelText('Slug').value).toBe('existing-slug')
    })
  })

  it('stops auto-generating slug once user edits the slug field', async () => {
    renderForm({ isEdit: false })

    // Auto-gen fires once
    fireEvent.change(screen.getByLabelText('Title'), {
      target: { value: 'First Title' },
    })
    await waitFor(() =>
      expect(screen.getByLabelText('Slug').value).toBe('first-title'),
    )

    // User manually edits slug
    fireEvent.change(screen.getByLabelText('Slug'), {
      target: { value: 'my-custom-slug' },
    })

    // Typing a new title should NOT overwrite the manual slug
    fireEvent.change(screen.getByLabelText('Title'), {
      target: { value: 'Second Title' },
    })
    await waitFor(() =>
      expect(screen.getByLabelText('Slug').value).toBe('my-custom-slug'),
    )
  })

  it('clears category when department changes', async () => {
    const { getForm } = renderForm()

    // Pre-set department + category programmatically
    act(() => {
      getForm().setFieldsValue({ department: 'Bags', category: 'Tote' })
    })

    expect(getForm().getFieldValue('category')).toBe('Tote')

    // Change department via the DOM — triggers onValuesChange → clears category
    const departmentItem = screen
      .getByText('Department')
      .closest('.ant-form-item')
    fireEvent.mouseDown(departmentItem.querySelector('.ant-select'))

    const wallets = await screen.findByTitle('Wallets')
    fireEvent.click(wallets)

    expect(getForm().getFieldValue('category')).toBeUndefined()
  })

  it('filters category options by selected department', async () => {
    renderForm()

    // Select Bags via UI so Form.useWatch updates the category options
    const departmentItem = screen
      .getByText('Department')
      .closest('.ant-form-item')
    fireEvent.mouseDown(departmentItem.querySelector('.ant-select'))
    fireEvent.click(await screen.findByTitle('Bags'))

    // Wait for category select to become enabled, then open it
    await waitFor(() => {
      const catSelect = screen
        .getByText('Category')
        .closest('.ant-form-item')
        .querySelector('.ant-select')
      expect(catSelect.className.includes('ant-select-disabled')).toBe(false)
    })

    const categoryItem = screen
      .getByText('Category')
      .closest('.ant-form-item')
    fireEvent.mouseDown(categoryItem.querySelector('.ant-select'))

    // Bags categories should be visible; Wallets categories should not
    expect(await screen.findByTitle('Tote')).toBeInTheDocument()
    expect(await screen.findByTitle('Crossbody')).toBeInTheDocument()
    expect(screen.queryByTitle('Bifold')).not.toBeInTheDocument()
  })
})
