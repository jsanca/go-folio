import { useState } from "react";
import { Table, Tag, InputNumber, Button, Space, message } from "antd";
import {
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
} from "@ant-design/icons";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../lib/useAuth";

const GATEWAY =
  import.meta.env.VITE_PUBLIC_GATEWAY_URL ?? "http://localhost:8090";

async function adjustStock(token, sku, delta) {
  const resp = await fetch(`${GATEWAY}/admin/inventory/${sku}`, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ delta, reason: "manual adjustment" }),
  });
  if (!resp.ok) throw new Error(`PUT /admin/inventory/${sku} → ${resp.status}`);
  return resp.json();
}

const STATUS_COLOR = {
  IN_STOCK: "green",
  LOW_STOCK: "orange",
  OUT_OF_STOCK: "red",
};

const STATUS_COLOR_LABEL = {
  IN_STOCK: "In Stock",
  LOW_STOCK: "Low Stock",
  OUT_OF_STOCK: "Out of Stock",
};

const STATUS_ICON = {
  IN_STOCK: <CheckCircleOutlined />,
  LOW_STOCK: <WarningOutlined />,
  OUT_OF_STOCK: <CloseCircleOutlined />,
};

function formatPrice(amountCents, currency) {
  return new Intl.NumberFormat("es-CR", {
    style: "currency",
    currency: currency ?? "CRC",
  }).format((amountCents ?? 0) / 100);
}

// EditableAvailable renders an inline-editable available stock count.
// Clicking opens an InputNumber; Enter confirms, Escape cancels.
// On confirm the component computes the delta and calls PUT /admin/inventory/{sku}.
function EditableAvailable({ sku, available }) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(available);
  const [loading, setLoading] = useState(false);
  const { token } = useAuth();
  const queryClient = useQueryClient();

  const startEditing = () => {
    setValue(available);
    setEditing(true);
  };

  const confirm = async () => {
    const delta = value - available;
    if (delta === 0) {
      setEditing(false);
      return;
    }
    setLoading(true);
    try {
      await adjustStock(token, sku, delta);
      queryClient.invalidateQueries({ queryKey: ["admin-products"] });
      setEditing(false);
      message.success(`Stock updated for ${sku}`, 3);
    } catch {
      message.error("Failed to update stock", 3);
    } finally {
      setLoading(false);
    }
  };

  const cancel = () => {
    setValue(available);
    setEditing(false);
  };

  if (!editing) {
    return (
      <span
        role="button"
        tabIndex={0}
        onClick={startEditing}
        onKeyDown={(e) => {
          if (e.key === "Enter") startEditing();
        }}
        style={{ cursor: "pointer" }}
      >
        {available}
      </span>
    );
  }

  return (
    <Space size="small">
      <InputNumber
        min={0}
        keyboard={true}
        controls={false}
        value={value}
        onChange={(v) => setValue(v ?? available)}
        onKeyDown={(e) => {
          if (e.key === "Enter") confirm();
          if (e.key === "Escape") cancel();
        }}
        disabled={loading}
        autoFocus
        size="small"
        style={{ width: 72 }}
      />
      <Button
        size="small"
        onClick={() => setValue((v) => (v ?? available) + 10)}
        disabled={loading}
      >
        +10
      </Button>
    </Space>
  );
}

const columns = [
  { title: "SKU", dataIndex: "sku", key: "sku" },
  { title: "Color", dataIndex: "colorName", key: "colorName" },
  {
    title: "Price",
    key: "price",
    render: (_, v) => formatPrice(v.retailPrice?.amountCents, v.currency),
  },
  {
    title: "Available",
    key: "available",
    render: (_, v) => (
      <EditableAvailable sku={v.sku} available={v.stock?.available ?? 0} />
    ),
  },
  {
    title: "Status",
    key: "status",
    render: (_, v) => {
      const status = v.stock?.stockStatus ?? "OUT_OF_STOCK";
      return (
        <Tag
          icon={STATUS_ICON[status]}
          color={STATUS_COLOR[status] ?? "default"}
        >
          {STATUS_COLOR_LABEL[status]}
        </Tag>
      );
    },
  },
];

// VariantStockTable renders a compact table of variants with live stock info.
export default function VariantStockTable({ variants }) {
  return (
    <Table
      columns={columns}
      dataSource={variants}
      rowKey="sku"
      pagination={false}
      size="small"
    />
  );
}
