import { Button, Card, Space, Typography, theme } from 'antd'
import { UserOutlined } from '@ant-design/icons'
import keycloak from '../lib/keycloak'

const { Title, Text } = Typography

export default function Login() {
  const { token } = theme.useToken()

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        background: token.colorBgLayout,
      }}
    >
      <Card style={{ width: 380, textAlign: 'center' }}>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div>
            <Title level={2} style={{ margin: '0 0 4px' }}>
              Folio Admin
            </Title>
            <Text type="secondary">Catalog &amp; inventory management</Text>
          </div>
          <Button
            type="primary"
            size="large"
            block
            icon={<UserOutlined />}
            onClick={() => keycloak.login()}
          >
            Sign in
          </Button>
        </Space>
      </Card>
    </div>
  )
}
