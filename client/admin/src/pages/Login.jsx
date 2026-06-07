import { Button, Card, Space, Typography, theme } from 'antd'
import { UserOutlined } from '@ant-design/icons'
import keycloak from '../lib/keycloak'
import WalletMosaicAnimation from './WalletMosaicAnimation'
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
          <WalletMosaicAnimation />
        </Space>
      </Card>
    </div>
  )
}
