import { Table, Tag, Typography } from 'antd'

const columns = [
  {
    title: 'Code',
    dataIndex: 'code',
    key: 'code',
  },
  {
    title: 'Title',
    dataIndex: 'title',
    key: 'title',
  },
  {
    title: 'Variants',
    dataIndex: 'variants',
    key: 'variants',
    align: 'right',
  },
  {
    title: 'Active',
    dataIndex: 'active',
    key: 'active',
    render: (active) => (
      <Tag color={active ? 'green' : 'default'}>{active ? 'Yes' : 'No'}</Tag>
    ),
  },
]

const data = [
  { key: '1', code: 'BAG-001',    title: 'Leather Tote Bag',     variants: 3, active: true  },
  { key: '2', code: 'BELT-001',   title: 'Classic Leather Belt',  variants: 5, active: true  },
  { key: '3', code: 'WALLET-001', title: 'Bifold Wallet',          variants: 2, active: false },
  { key: '4', code: 'CARD-001',   title: 'Card Holder',            variants: 4, active: true  },
]

export default function Products() {
  return (
    <>
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        Products
      </Typography.Title>
      <Table columns={columns} dataSource={data} pagination={false} />
    </>
  )
}
