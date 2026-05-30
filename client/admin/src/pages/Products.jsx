import { Table, Tag, Typography, Alert, Spin } from "antd";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../lib/useAuth";

const GATEWAY =
  import.meta.env.VITE_PUBLIC_GATEWAY_URL ?? "http://localhost:8090";

const columns = [
  {
    title: "Code",
    dataIndex: "productCode",
    key: "productCode",
  },
  {
    title: "Title",
    dataIndex: "title",
    key: "title",
  },
  {
    title: "Variants",
    key: "variants",
    align: "right",
    render: (_, record) => record.variants?.length ?? 0,
  },
  {
    title: "Active",
    key: "active",
    render: (_, record) => {
      const vs = record.variants ?? [];
      const active = vs.length > 0 && vs.every((v) => v.active);
      return (
        <Tag color={active ? "green" : "default"}>{active ? "Yes" : "No"}</Tag>
      );
    },
  },
];

async function fetchAdminProducts(token) {
  const resp = await fetch(`${GATEWAY}/admin/products`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!resp.ok) {
    const err = new Error(`GET /admin/products → ${resp.status}`);
    console.error(err);
    throw err;
  }
  return resp.json();
}

export default function Products() {
  const { authenticated, token } = useAuth();
  const { data, isPending, isError, error } = useQuery({
    queryKey: ["admin-products"],
    queryFn: () => fetchAdminProducts(token),
    enabled: authenticated && !!token,
  });

  if (isPending) {
    return (
      <div style={{ textAlign: "center", padding: 48 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <>
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        Products
      </Typography.Title>
      {isError && (
        <Alert
          type="error"
          message="Failed to load products"
          description={error.message}
          style={{ marginBottom: 16 }}
        />
      )}
      <Table
        columns={columns}
        dataSource={data ?? []}
        rowKey="id"
        pagination={false}
      />
    </>
  );
}
