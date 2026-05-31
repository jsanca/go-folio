import { useRef, useState } from 'react'
import { Alert, Form, Input, Modal, message } from 'antd'
import Draggable from 'react-draggable'
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

// AdjustStockForm renders a draggable modal form for adjusting stock for a single SKU.
// `sku`       — the SKU to adjust; shown in the modal title.
// `open`      — controls modal visibility.
// `onClose`   — called when the modal is dismissed or the adjustment succeeds.
// `onSuccess` — optional; called with the SKU after a successful adjustment.
export default function AdjustStockForm({ sku, open, onClose, onSuccess }) {
  const { token } = useAuth()
  const [form] = Form.useForm()
  const queryClient = useQueryClient()
  const [inlineError, setInlineError] = useState(null)
  const draggleRef = useRef(null)

  const mutation = useMutation({
    mutationFn: (values) =>
      apiAdjustStock(token, sku, Number(values.delta), values.reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-inventory'] })
      form.resetFields()
      setInlineError(null)
      message.success(`Stock updated for ${sku}`, 3)
      document.querySelector(`tr[data-row-key="${sku}"]`)
        ?.scrollIntoView?.({ behavior: 'smooth', block: 'center' })
      onSuccess?.(sku)
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
      title={
        <div style={{ cursor: 'move' }}>
          Adjust Stock — {sku}
        </div>
      }
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText="Adjust"
      confirmLoading={mutation.isPending}
      destroyOnHidden
      modalRender={(modal) => (
        <Draggable
          nodeRef={draggleRef}
          handle=".ant-modal-header"
          bounds="body"
        >
          <div ref={draggleRef}>{modal}</div>
        </Draggable>
      )}
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
