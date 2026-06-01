import { useState } from 'react'
import { Modal, Form, Input, InputNumber, Select, Switch, message } from 'antd'
import { useQueryClient } from '@tanstack/react-query'
import { useAuth } from '../lib/useAuth'

const GATEWAY =
  import.meta.env.VITE_PUBLIC_GATEWAY_URL ?? 'http://localhost:8090'

async function apiAddVariant(token, productId, values) {
  const resp = await fetch(`${GATEWAY}/admin/products/${productId}/variants`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(values),
  })
  if (!resp.ok)
    throw new Error(`POST /admin/products/${productId}/variants → ${resp.status}`)
  return resp.json()
}

function slugify(text) {
  return (text ?? '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '')
}

const CURRENCY_OPTIONS = [
  { value: 'CRC', label: 'CRC' },
  { value: 'USD', label: 'USD' },
]

// VariantForm renders a modal form for adding a variant to a product.
// `productId` — the product to attach the variant to.
// `open`      — controls modal visibility.
// `onClose`   — called when the modal should be closed.
export default function VariantForm({ productId, open, onClose }) {
  const [form] = Form.useForm()
  const { token } = useAuth()
  const queryClient = useQueryClient()
  const [loading, setLoading] = useState(false)

  const handleValuesChange = (changed) => {
    if ('colorName' in changed) {
      form.setFieldValue('colorSlug', slugify(changed.colorName))
    }
  }

  const handleOk = async () => {
    let values
    try {
      values = await form.validateFields()
    } catch {
      return // validation errors are shown inline
    }
    setLoading(true)
    try {
      await apiAddVariant(token, productId, values)
      queryClient.invalidateQueries({ queryKey: ['admin-products'] })
      message.success('Variant added')
      form.resetFields()
      onClose()
    } catch (err) {
      message.error(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = () => {
    form.resetFields()
    onClose()
  }

  return (
    <Modal
      title="Add Variant"
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText="Add Variant"
      confirmLoading={loading}
      destroyOnHidden
    >
      <Form form={form} layout="vertical" onValuesChange={handleValuesChange}>
        <Form.Item
          name="sku"
          label="SKU"
          rules={[{ required: true, message: 'SKU is required' }]}
        >
          <Input />
        </Form.Item>

        <Form.Item name="colorName" label="Color Name">
          <Input />
        </Form.Item>

        <Form.Item name="colorSlug" label="Color Slug">
          <Input />
        </Form.Item>

        <Form.Item
          name="primaryColorHex"
          label="Primary Color"
          initialValue="#000000"
        >
          <Input type="color" />
        </Form.Item>

        <Form.Item
          name="retailPriceCents"
          label="Retail Price (cents)"
          rules={[{ required: true, message: 'Price is required' }]}
        >
          <InputNumber min={0} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="currency" label="Currency" initialValue="CRC">
          <Select options={CURRENCY_OPTIONS} />
        </Form.Item>

        <Form.Item
          name="active"
          label="Active"
          valuePropName="checked"
          initialValue={true}
        >
          <Switch />
        </Form.Item>
      </Form>
    </Modal>
  )
}
