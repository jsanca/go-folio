import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import ProtectedRoute from './ProtectedRoute'

// ProtectedRoute checks keycloak.hasRealmRole directly, not useAuth.
const mockKeycloak = vi.hoisted(() => ({
  hasRealmRole: vi.fn(),
  logout: vi.fn(),
}))

vi.mock('../lib/keycloak', () => ({ default: mockKeycloak }))

beforeEach(() => {
  vi.clearAllMocks()
})

function renderRoute(isAdmin) {
  mockKeycloak.hasRealmRole.mockReturnValue(isAdmin)
  render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route element={<ProtectedRoute />}>
          <Route path="/" element={<div>Protected Content</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  )
}

describe('ProtectedRoute', () => {
  it('shows 403 result when user does not have the admin role', () => {
    renderRoute(false)

    expect(screen.getByText('403')).toBeInTheDocument()
    expect(screen.getByText(/does not have the admin role/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument()
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
  })

  it('renders children when user has the admin role', () => {
    renderRoute(true)

    expect(screen.getByText('Protected Content')).toBeInTheDocument()
    expect(screen.queryByText('403')).not.toBeInTheDocument()
  })

  it('Sign out button calls keycloak.logout', () => {
    renderRoute(false)

    fireEvent.click(screen.getByRole('button', { name: 'Sign out' }))

    expect(mockKeycloak.logout).toHaveBeenCalledOnce()
  })
})
