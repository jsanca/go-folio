import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useAuth } from './useAuth'

// vi.mock is hoisted to the top of the file, so the factory cannot reference
// a const declared below it. vi.hoisted() runs in the same hoist pass and
// returns a value that is safe to reference inside vi.mock factories.
const mockKeycloak = vi.hoisted(() => ({
  authenticated: false,
  token: undefined,
  onAuthSuccess: undefined,
  onAuthLogout: undefined,
}))

vi.mock('./keycloak', () => ({ default: mockKeycloak }))

beforeEach(() => {
  mockKeycloak.authenticated = false
  mockKeycloak.token = undefined
  mockKeycloak.onAuthSuccess = undefined
  mockKeycloak.onAuthLogout = undefined
})

describe('useAuth', () => {
  it('returns authenticated: false before Keycloak fires', () => {
    const { result } = renderHook(() => useAuth())
    expect(result.current.authenticated).toBe(false)
    expect(result.current.token).toBeUndefined()
  })

  it('returns authenticated: true and token after onAuthSuccess fires', () => {
    const { result } = renderHook(() => useAuth())

    act(() => {
      mockKeycloak.authenticated = true
      mockKeycloak.token = 'eyJhbGciOiJSUzI1NiJ9.test'
      mockKeycloak.onAuthSuccess()
    })

    expect(result.current.authenticated).toBe(true)
    expect(result.current.token).toBe('eyJhbGciOiJSUzI1NiJ9.test')
  })

  it('returns authenticated: false after onAuthLogout fires', () => {
    mockKeycloak.authenticated = true
    mockKeycloak.token = 'eyJhbGciOiJSUzI1NiJ9.test'

    const { result } = renderHook(() => useAuth())
    expect(result.current.authenticated).toBe(true)

    act(() => {
      mockKeycloak.authenticated = false
      mockKeycloak.token = undefined
      mockKeycloak.onAuthLogout()
    })

    expect(result.current.authenticated).toBe(false)
    expect(result.current.token).toBeUndefined()
  })
})
