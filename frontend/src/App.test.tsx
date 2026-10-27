import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderApp } from './test/render'

describe('the app shell', () => {
  it('names the dealership and the user', async () => {
    renderApp()

    expect(await screen.findByText('Northside Motors')).toBeInTheDocument()
    expect(screen.getByText(/alice/)).toBeInTheDocument()
  })

  it('has a way home from any page', async () => {
    renderApp({ route: '/nowhere' })

    expect(await screen.findByRole('heading', { name: 'Page not found' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Go to the inventory' })).toHaveAttribute('href', '/')
  })
})
