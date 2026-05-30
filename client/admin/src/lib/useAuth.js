import { useState, useEffect } from 'react'
import keycloak from './keycloak'

// Returns reactive { authenticated, token } that updates when Keycloak fires
// onAuthSuccess or onAuthLogout — safe regardless of when init() completes.
export function useAuth() {
  const [state, setState] = useState({
    authenticated: keycloak.authenticated ?? false,
    token: keycloak.token,
  })

  useEffect(() => {
    const onSuccess = () =>
      setState({ authenticated: true, token: keycloak.token })
    const onLogout = () =>
      setState({ authenticated: false, token: undefined })

    keycloak.onAuthSuccess = onSuccess
    keycloak.onAuthLogout = onLogout

    // Sync in case init already completed before this effect ran
    if (keycloak.authenticated) {
      setState({ authenticated: true, token: keycloak.token })
    }

    return () => {
      if (keycloak.onAuthSuccess === onSuccess) keycloak.onAuthSuccess = undefined
      if (keycloak.onAuthLogout === onLogout) keycloak.onAuthLogout = undefined
    }
  }, [])

  return state
}
