import { useState } from 'react'
import {
  Table,
  Tag,
  Typography,
  Alert,
  Spin,
  Button,
  Modal,
  Space,
  Form,
  message,
} from 'antd'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircleOutlined, StopOutlined } from '@ant-design/icons'
import { useAuth } from '../lib/useAuth'
import ProductForm from '../components/ProductForm'
import VariantForm from '../components/VariantForm'
import VariantStockTable from '../components/VariantStockTable'

const GATEWAY =
  import.meta.env.VITE_PUBLIC_GATEWAY_URL ?? 'http://localhost:8090'

async function fetchAdminProducts(token) {
  const resp = await fetch(`${GATEWAY}/admin/products`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!resp.ok) {
    const err = new Error(`GET /admin/products → ${resp.status}`)
    console.error(err)
    throw err
  }
  return resp.json()
}

async function apiCreateProduct(token, values) {
  const resp = await fetch(`${GATEWAY}/admin/products`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(values),
  })
  if (!resp.ok) throw new Error(`POST /admin/products → ${resp.status}`)
  return resp.json()
}

async function apiUpdateProduct(token, id, values) {
  const resp = await fetch(`${GATEWAY}/admin/products/${id}`, {
    method: 'PATCH',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(values),
  })
  if (!resp.ok) throw new Error(`PATCH /admin/products/${id} → ${resp.status}`)
  return resp.json()
}

async function apiDeleteProduct(token, id) {
  const resp = await fetch(`${GATEWAY}/admin/products/${id}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!resp.ok) throw new Error(`DELETE /admin/products/${id} → ${resp.status}`)
}

export default function Products() {
  const { authenticated, token } = useAuth()
  const queryClient = useQueryClient()
  const [form] = Form.useForm()
  const [modalOpen, setModalOpen] = useState(false)
  const [editingProduct, setEditingProduct] = useState(null)
  const [variantProductId, setVariantProductId] = useState(null)

  const { data, isPending, isError, error } = useQuery({
    queryKey: ['admin-products'],
    queryFn: () => fetchAdminProducts(token),
    enabled: authenticated && !!token,
  })

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ['admin-products'] })

  const createMutation = useMutation({
    mutationFn: (values) => apiCreateProduct(token, values),
    onSuccess: () => {
      invalidate()
      setModalOpen(false)
      message.success('Product created')
    },
    onError: (err) => message.error(err.message),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, ...values }) => apiUpdateProduct(token, id, values),
    onSuccess: () => {
      invalidate()
      setModalOpen(false)
      message.success('Product updated')
    },
    onError: (err) => message.error(err.message),
  })

  const deleteMutation = useMutation({
    mutationFn: (id) => apiDeleteProduct(token, id),
    onSuccess: () => {
      invalidate()
      message.success('Product deleted')
    },
    onError: (err) => message.error(err.message),
  })

  const openCreate = () => {
    form.resetFields()
    setEditingProduct(null)
    setModalOpen(true)
  }

  const openEdit = (record) => {
    form.setFieldsValue({
      productCode: record.productCode,
      title: record.title,
      slug: record.slug,
      shortDescription: record.shortDescription ?? '',
      department: record.department,
      category: record.category,
      active: record.active ?? true,
    })
    setEditingProduct(record)
    setModalOpen(true)
  }

  const handleModalOk = () => {
    form.validateFields().then((values) => {
      if (editingProduct) {
        updateMutation.mutate({ id: editingProduct.id, ...values })
      } else {
        createMutation.mutate(values)
      }
    })
  }

  const handleDelete = (record) => {
    Modal.confirm({
      title: 'Delete product?',
      content: `"${record.title}" will be permanently removed.`,
      okText: 'Delete',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: () => deleteMutation.mutate(record.id),
    })
  }

  const columns = [
    {
      title: '',
      key: 'thumbnail',
      width: 56,
      render: (_, record) => {
        const color = record.variants?.[0]?.primaryColorHex ?? '#e5e5e5'
        return (
          <div style={{
            width: 40,
            height: 40,
            borderRadius: 4,
            backgroundColor: color,
            border: '1px solid #eee',
          }} />
        )
      },
    },
    {
      title: 'Code',
      dataIndex: 'productCode',
      key: 'productCode',
    },
    {
      title: 'Title',
      dataIndex: 'title',
      key: 'title',
    },
    {
      title: 'Variants',
      key: 'variants',
      align: 'right',
      render: (_, record) => record.variants?.length ?? 0,
    },
    {
      title: 'Active',
      key: 'active',
      render: (_, record) => {
        const active = record.active
        return (
          <Tag icon={active ? <CheckCircleOutlined /> : <StopOutlined />} color={active ? 'green' : 'default'}>
            {active ? 'Active' : 'Inactive'}
          </Tag>
        )
      },
    },
    {
      title: '',
      key: 'actions',
      align: 'right',
      render: (_, record) => (
        <Space>
          <Button size="small" onClick={() => openEdit(record)}>
            Edit
          </Button>
          <Button size="small" onClick={() => setVariantProductId(record.id)}>
            Add Variant
          </Button>
          <Button size="small" danger onClick={() => handleDelete(record)}>
            Delete
          </Button>
        </Space>
      ),
    },
  ]

  const isMutating = createMutation.isPending || updateMutation.isPending

  return (
    <>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <div>
          <Typography.Title level={4} style={{ margin: 0 }}>
            Products
          </Typography.Title>
          {data && (
            <Space style={{ marginTop: 4 }}>
              <Tag>{data.length} total</Tag>
              <Tag color="green">{data.filter((p) => p.active).length} active</Tag>
              <Tag color="default">{data.filter((p) => !p.active).length} inactive</Tag>
            </Space>
          )}
        </div>
        <Button type="primary" onClick={openCreate}>
          Create Product
        </Button>
      </div>

      {isError && (
        <Alert
          type="error"
          message="Failed to load products"
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
          rowKey="id"
          pagination={false}
          expandable={{
            expandedRowRender: (record) => (
              <VariantStockTable variants={record.variants} />
            ),
            rowExpandable: (record) => record.variants?.length > 0,
          }}
        />
      )}

      <VariantForm
        productId={variantProductId}
        open={variantProductId !== null}
        onClose={() => setVariantProductId(null)}
      />

      <Modal
        title={editingProduct ? 'Edit Product' : 'Create Product'}
        open={modalOpen}
        onOk={handleModalOk}
        onCancel={() => setModalOpen(false)}
        okText={editingProduct ? 'Save' : 'Create'}
        confirmLoading={isMutating}
        destroyOnHidden
      >
        <ProductForm
          key={editingProduct?.id ?? 'new'}
          form={form}
          isEdit={!!editingProduct}
        />
      </Modal>
    </>
  )
}
