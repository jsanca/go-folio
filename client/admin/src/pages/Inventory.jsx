import { Table, Tag, Typography } from 'antd'

const STATUS_COLOR = {
  IN_STOCK:     'green',
  LOW_STOCK:    'orange',
  OUT_OF_STOCK: 'red',
}

const columns = [
  {
    title: 'SKU',
    dataIndex: 'sku',
    key: 'sku',
  },
  {
    title: 'Available',
    dataIndex: 'available',
    key: 'available',
    align: 'right',
  },
  {
    title: 'Reserved',
    dataIndex: 'reserved',
    key: 'reserved',
    align: 'right',
  },
  {
    title: 'Status',
    dataIndex: 'status',
    key: 'status',
    render: (status) => (
      <Tag color={STATUS_COLOR[status]}>
        {status.replace(/_/g, ' ')}
      </Tag>
    ),
  },
]

const data = [
  { key: '1', sku: 'BAG-001-BRN-M',   available: 12, reserved: 2, status: 'IN_STOCK'     },
  { key: '2', sku: 'BAG-001-BLK-M',   available:  3, reserved: 1, status: 'LOW_STOCK'    },
  { key: '3', sku: 'BELT-001-BRN-32', available:  0, reserved: 0, status: 'OUT_OF_STOCK' },
  { key: '4', sku: 'BELT-001-BRN-34', available:  8, reserved: 2, status: 'IN_STOCK'     },
  { key: '5', sku: 'WALLET-001-BRN',  available:  4, reserved: 0, status: 'LOW_STOCK'    },
]

export default function Inventory() {
  return (
    <>
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        Inventory
      </Typography.Title>
      <Table columns={columns} dataSource={data} pagination={false} />
    </>
  )
}
