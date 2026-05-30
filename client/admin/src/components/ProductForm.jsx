import { useRef } from 'react'
import { Form, Input, Select, Switch } from 'antd'
import { DEPARTMENTS, CATEGORIES } from '../config/catalogs'

export function slugify(text) {
  return (text ?? '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '')
}

// ProductForm renders the product create/edit form fields.
// `form`   — Ant Design form instance created by the parent with Form.useForm().
// `isEdit` — when true the slug field starts in "manually edited" state so
//            typing in Title will not overwrite an existing slug.
export default function ProductForm({ form, isEdit = false }) {
  // Track whether the user has manually touched the slug field.
  // Initialised to true in edit mode so we never auto-overwrite an existing slug.
  const slugEditedRef = useRef(isEdit)

  const handleValuesChange = (changed) => {
    if ('title' in changed && !slugEditedRef.current) {
      form.setFieldValue('slug', slugify(changed.title))
    }
    if ('slug' in changed) {
      slugEditedRef.current = true
    }
    if ('department' in changed) {
      form.setFieldValue('category', undefined)
    }
  }

  const department = Form.useWatch('department', form)
  const departmentOptions = DEPARTMENTS.map((d) => ({ value: d, label: d }))
  const categoryOptions = (CATEGORIES[department] ?? []).map((c) => ({
    value: c,
    label: c,
  }))

  return (
    <Form form={form} layout="vertical" onValuesChange={handleValuesChange}>
      <Form.Item
        name="productCode"
        label="Product Code"
        rules={[{ required: true, message: 'Product code is required' }]}
      >
        <Input />
      </Form.Item>

      <Form.Item
        name="title"
        label="Title"
        rules={[{ required: true, message: 'Title is required' }]}
      >
        <Input />
      </Form.Item>

      <Form.Item
        name="slug"
        label="Slug"
        rules={[{ required: true, message: 'Slug is required' }]}
      >
        <Input />
      </Form.Item>

      <Form.Item name="shortDescription" label="Short Description">
        <Input.TextArea rows={3} />
      </Form.Item>

      <Form.Item name="department" label="Department">
        <Select options={departmentOptions} allowClear placeholder="Select department" />
      </Form.Item>

      <Form.Item name="category" label="Category">
        <Select
          options={categoryOptions}
          allowClear
          disabled={!department}
          placeholder="Select category"
        />
      </Form.Item>

      <Form.Item name="active" label="Active" valuePropName="checked" initialValue={true}>
        <Switch />
      </Form.Item>
    </Form>
  )
}
