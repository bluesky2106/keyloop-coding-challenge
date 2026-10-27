import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { tokenStore } from '../api/tokenStore'
import { renderApp } from '../test/render'
import { apiError, server, url } from '../test/server'

async function signIn(
  user: ReturnType<typeof renderApp>['user'],
  username: string,
  password: string,
) {
  await user.type(screen.getByLabelText('Username'), username)
  await user.type(screen.getByLabelText('Password'), password)
  await user.click(screen.getByRole('button', { name: 'Sign in' }))
}

describe('signing in', () => {
  it('sends a signed-out visitor to the login page', async () => {
    renderApp({ token: null })

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
    expect(screen.getByTestId('location')).toHaveTextContent('/login')
  })

  it('stores the token and opens the inventory after a successful login', async () => {
    const { user } = renderApp({ route: '/login', token: null })

    await signIn(user, 'alice', 'secret')

    expect(await screen.findByRole('heading', { name: 'Inventory' })).toBeInTheDocument()
    expect(tokenStore.get()).toBe('token-1')
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/)
  })

  it('returns to the page the visitor first asked for', async () => {
    const { user } = renderApp({ route: '/?aging=true&sort=price_asc', token: null })
    await screen.findByRole('heading', { name: 'Sign in' })

    await signIn(user, 'alice', 'secret')

    await screen.findByRole('heading', { name: 'Inventory' })
    expect(screen.getByTestId('location')).toHaveTextContent('/?aging=true&sort=price_asc')
  })

  it('says so, and stays signed out, when the credentials are wrong', async () => {
    const { user } = renderApp({ route: '/login', token: null })

    await signIn(user, 'alice', 'wrong')

    expect(await screen.findByRole('alert')).toHaveTextContent('Invalid username or password.')
    expect(tokenStore.get()).toBeNull()
    expect(screen.getByRole('button', { name: 'Sign in' })).toBeEnabled()
  })

  it('explains a rejected request', async () => {
    server.use(
      http.post(url('/auth/login'), () => apiError(400, 'validation_error', 'bad request')),
    )
    const { user } = renderApp({ route: '/login', token: null })

    await signIn(user, 'alice', 'secret')

    expect(await screen.findByRole('alert')).toHaveTextContent('Enter your username and password.')
  })

  it('shows a generic message for a server failure', async () => {
    server.use(http.post(url('/auth/login'), () => new HttpResponse(null, { status: 500 })))
    const { user } = renderApp({ route: '/login', token: null })

    await signIn(user, 'alice', 'secret')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Something went wrong. Please try again.',
    )
  })

  it('shows a generic message when the network is down', async () => {
    server.use(http.post(url('/auth/login'), () => HttpResponse.error()))
    const { user } = renderApp({ route: '/login', token: null })

    await signIn(user, 'alice', 'secret')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Something went wrong. Please try again.',
    )
  })

  it('does not show the login form to someone already signed in', async () => {
    renderApp({ route: '/login' })

    expect(await screen.findByRole('heading', { name: 'Inventory' })).toBeInTheDocument()
  })
})

describe('signing out', () => {
  it('clears the session and shows the login page', async () => {
    const { user } = renderApp()
    await screen.findByRole('heading', { name: 'Inventory' })

    await user.click(screen.getByRole('button', { name: 'Sign out' }))

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
    expect(tokenStore.get()).toBeNull()
  })

  it('does not show the previous user their data after the next sign-in', async () => {
    const { user, client } = renderApp()
    await screen.findByText('Toyota Corolla')
    expect(client.getQueryCache().getAll().length).toBeGreaterThan(0)

    await user.click(screen.getByRole('button', { name: 'Sign out' }))

    await waitFor(() => {
      expect(client.getQueryCache().getAll()).toHaveLength(0)
    })
  })

  it('signs the user out when the API rejects the token', async () => {
    server.use(http.get(url('/vehicles'), () => apiError(401, 'unauthorized', 'expired')))
    renderApp()

    expect(await screen.findByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
    expect(tokenStore.get()).toBeNull()
  })
})
