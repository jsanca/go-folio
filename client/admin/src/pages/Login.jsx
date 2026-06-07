import { Button, Card, Space, Typography, theme } from 'antd'
import { UserOutlined } from '@ant-design/icons'
import keycloak from '../lib/keycloak'
import './Login.css'

const { Title, Text } = Typography

export default function Login() {
  const { token } = theme.useToken()

  return (
    <div
      className="login-shell"
      style={{ background: token.colorBgLayout }}
    >
      <Card className="login-card">
        <div className="wallet-intro" aria-hidden="true">
          <span className="wallet-piece wallet-piece-top-left-a" />
          <span className="wallet-piece wallet-piece-top-left-b" />
          <span className="wallet-piece wallet-piece-top-right-a" />
          <span className="wallet-piece wallet-piece-top-right-b" />
          <span className="wallet-piece wallet-piece-bottom-left-a" />
          <span className="wallet-piece wallet-piece-bottom-left-b" />
          <span className="wallet-piece wallet-piece-bottom-right-a" />
          <span className="wallet-piece wallet-piece-bottom-right-b" />
          <span className="wallet-stitch" />
        </div>
        <Space direction="vertical" size="large" className="login-content">
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
