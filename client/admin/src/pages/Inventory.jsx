import { useState } from 'react'
import { Alert, Button, Spin, Table, Tag, Typography } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from '../lib/useAuth'
import AdjustStockForm from '../components/AdjustStockForm'

const GATEWAY = import.meta.env.VITE_PUBLIC_GATEWAY_URL ?? 'http://localhost:8090'

const STATUS_COLOR = {
  IN_STOCK:     'green',
  LOW_STOCK:    'orange',
  OUT_OF_STOCK: 'red',
}

async function fetchInventory(token) {
  const resp = await fetch(`${GATEWAY}/admin/inventory`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!resp.ok) throw new Error(`GET /admin/inventory → ${resp.status}`)
  return resp.json()
}

export default function Inventory() {
  const { authenticated, token } = useAuth()
  const [adjustSKU, setAdjustSKU] = useState(null)

  const { data, isPending, isError, error } = useQuery({
    queryKey: ['admin-inventory'],
    queryFn: () => fetchInventory(token),
    enabled: authenticated && !!token,
  })

  const columns = [
    { title: 'SKU',       dataIndex: 'sku',       key: 'sku' },
    { title: 'Available', dataIndex: 'available', key: 'available', align: 'right' },
    { title: 'Reserved',  dataIndex: 'reserved',  key: 'reserved',  align: 'right' },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={STATUS_COLOR[status]}>{status.replace(/_/g, ' ')}</Tag>
      ),
    },
    {
      title: '',
      key: 'actions',
      align: 'right',
      render: (_, record) => (
        <Button size="small" onClick={() => setAdjustSKU(record.sku)}>
          Adjust
        </Button>
      ),
    },
  ]

  return (
    <>
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        Inventory
      </Typography.Title>

      {isError && (
        <Alert
          type="error"
          message="Failed to load inventory"
          description={error.message}
          style={{ marginBottom: 16 }}
        />
      )}

      {isPending ? (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : (
        <Table
          columns={columns}
          dataSource={data ?? []}
          rowKey="sku"
          pagination={false}
        />
      )}

      <AdjustStockForm
        sku={adjustSKU}
        open={!!adjustSKU}
        onClose={() => setAdjustSKU(null)}
      />
    </>
  )
}
