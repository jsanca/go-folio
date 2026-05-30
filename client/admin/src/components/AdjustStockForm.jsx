import { useState } from 'react'
import { Alert, Form, Input, Modal } from 'antd'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '../lib/useAuth'

const GATEWAY = import.meta.env.VITE_PUBLIC_GATEWAY_URL ?? 'http://localhost:8090'

async function apiAdjustStock(token, sku, delta, reason) {
  const resp = await fetch(`${GATEWAY}/admin/inventory/${sku}`, {
    method: 'PUT',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ delta, reason: reason ?? '' }),
  })
  if (resp.status === 422) {
    const err = new Error('Insufficient stock')
    err.status = 422
    throw err
  }
  if (resp.status === 404) {
    const err = new Error('SKU not found')
    err.status = 404
    throw err
  }
  if (!resp.ok) throw new Error(`PUT /admin/inventory/${sku} → ${resp.status}`)
  return resp.json()
}

// AdjustStockForm renders a modal form for adjusting stock for a single SKU.
// `sku`     — the SKU to adjust; the modal title shows this value.
// `open`    — controls modal visibility.
// `onClose` — called when the modal is dismissed or the adjustment succeeds.
export default function AdjustStockForm({ sku, open, onClose }) {
  const { token } = useAuth()
  const [form] = Form.useForm()
  const queryClient = useQueryClient()
  const [inlineError, setInlineError] = useState(null)

  const mutation = useMutation({
    mutationFn: (values) =>
      apiAdjustStock(token, sku, Number(values.delta), values.reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-inventory'] })
      form.resetFields()
      setInlineError(null)
      onClose()
    },
    onError: (err) => {
      setInlineError(err.message)
    },
  })

  const handleOk = () => {
    form.validateFields().then((values) => {
      setInlineError(null)
      mutation.mutate(values)
    })
  }

  const handleCancel = () => {
    form.resetFields()
    setInlineError(null)
    onClose()
  }

  return (
    <Modal
      title={`Adjust Stock — ${sku}`}
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText="Adjust"
      confirmLoading={mutation.isPending}
      destroyOnHidden
    >
      {inlineError && (
        <Alert type="error" message={inlineError} style={{ marginBottom: 16 }} />
      )}
      <Form form={form} layout="vertical">
        <Form.Item
          name="delta"
          label="Delta"
          rules={[{ required: true, message: 'Delta is required' }]}
        >
          <Input type="number" />
        </Form.Item>
        <Form.Item name="reason" label="Reason">
          <Input />
        </Form.Item>
      </Form>
    </Modal>
  )
}
