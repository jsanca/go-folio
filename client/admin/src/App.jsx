import { Routes, Route, Navigate, Link, useLocation } from 'react-router-dom'
import { Layout, Menu, Button, Typography, theme } from 'antd'
import {
  AppstoreOutlined,
  LogoutOutlined,
  UserOutlined,
} from '@ant-design/icons'
import keycloak from './lib/keycloak'
import ProtectedRoute from './components/ProtectedRoute'
import Login from './pages/Login'
import Products from './pages/Products'

const { Sider, Header, Content } = Layout

function AdminShell() {
  const location = useLocation()
  const { token } = theme.useToken()
  const username = keycloak.tokenParsed?.preferred_username ?? 'User'

  const menuItems = [
    {
      key: '/products',
      icon: <AppstoreOutlined />,
      label: <Link to="/products">Products</Link>,
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider>
        <div
          style={{
            padding: '16px 24px',
            color: 'rgba(255,255,255,.85)',
            fontWeight: 600,
            fontSize: 16,
            borderBottom: '1px solid rgba(255,255,255,.1)',
            marginBottom: 8,
          }}
        >
          Folio Admin
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
        />
      </Sider>

      <Layout>
        <Header
          style={{
            background: token.colorBgContainer,
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            gap: 8,
            padding: '0 24px',
          }}
        >
          <UserOutlined style={{ color: token.colorTextSecondary }} />
          <Typography.Text>{username}</Typography.Text>
          <Button
            type="text"
            icon={<LogoutOutlined />}
            onClick={() => keycloak.logout()}
          >
            Logout
          </Button>
        </Header>

        <Content
          style={{
            padding: 24,
            background: token.colorBgLayout,
          }}
        >
          <Routes>
            <Route index element={<Navigate to="/products" replace />} />
            <Route element={<ProtectedRoute />}>
              <Route path="/products" element={<Products />} />
            </Route>
            <Route path="*" element={<Navigate to="/products" replace />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  )
}

export default function App() {
  if (!keycloak.authenticated) {
    return <Login />
  }
  return <AdminShell />
}
