import { Outlet } from 'react-router-dom'
import { Result, Button } from 'antd'
import keycloak from '../lib/keycloak'

export default function ProtectedRoute() {
  if (!keycloak.hasRealmRole('admin')) {
    return (
      <Result
        status="403"
        title="403"
        subTitle="Your account does not have the admin role required to view this page."
        extra={
          <Button type="primary" onClick={() => keycloak.logout()}>
            Sign out
          </Button>
        }
      />
    )
  }
  return <Outlet />
}
