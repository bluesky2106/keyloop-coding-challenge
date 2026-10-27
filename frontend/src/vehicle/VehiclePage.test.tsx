import { screen, waitFor, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import type { VehicleAction } from '../api/types'
import { action, agingVehicle, ids } from '../test/fixtures'
import { renderApp } from '../test/render'
import { apiError, server, url } from '../test/server'

const agingPath = `/vehicles/${ids.agingVehicle}`
const freshPath = `/vehicles/${ids.freshVehicle}`

/** An actions endpoint backed by a list that POST appends to, like the real one. */
function statefulActions(initial: VehicleAction[] = []) {
  const history = [...initial]
  const posted: unknown[] = []
  server.use(
    http.get(url('/vehicles/:id/actions'), () =>
      HttpResponse.json({ items: [...history].reverse() }),
    ),
    http.post(url('/vehicles/:id/actions'), async ({ request }) => {
      const body = (await request.json()) as {
        action_type: VehicleAction['action_type']
        note: string | null
      }
      posted.push(body)
      const created = action({
        id: `a${String(history.length + 10)}`,
        ...body,
        created_at: '2026-04-01T09:30:00Z',
      })
      history.push(created)
      return HttpResponse.json(created, { status: 201 })
    }),
    http.get(url('/vehicles/:id'), () =>
      HttpResponse.json({ ...agingVehicle, current_action: history.at(-1) ?? null }),
    ),
  )
  return { posted }
}

describe('the vehicle page', () => {
  it('shows the vehicle in full', async () => {
    renderApp({ route: agingPath })

    expect(await screen.findByRole('heading', { name: /Toyota Corolla/ })).toBeInTheDocument()
    const details = screen.getByRole('region', { name: 'Vehicle details' })
    expect(within(details).getByText('1HGCM82633A004352')).toBeInTheDocument()
    expect(within(details).getByText('$24,500')).toBeInTheDocument()
    expect(within(details).getByText('15 Jan 2026')).toBeInTheDocument()
    expect(within(details).getByText(/120 days/)).toBeInTheDocument()
    expect(within(details).getByText(/Aging/)).toBeInTheDocument()
    expect(within(details).getByText('None logged')).toBeInTheDocument()
  })

  it('shows the current action and the history, newest first', async () => {
    const older = action({
      id: 'a1',
      action_type: 'hold',
      note: null,
      created_at: '2026-03-01T12:00:00Z',
    })
    const newer = action({ id: 'a2', action_type: 'transfer_planned', note: 'Send to Riverside' })
    server.use(
      http.get(url('/vehicles/:id'), () =>
        HttpResponse.json({ ...agingVehicle, current_action: newer }),
      ),
      http.get(url('/vehicles/:id/actions'), () => HttpResponse.json({ items: [newer, older] })),
    )
    renderApp({ route: agingPath })

    const history = await screen.findByRole('list')
    const items = within(history).getAllByRole('listitem')
    expect(items).toHaveLength(2)
    expect(within(items[0]).getByText('Transfer planned')).toBeInTheDocument()
    expect(within(items[0]).getByText('Send to Riverside')).toBeInTheDocument()
    expect(within(items[0]).getByText(/alice/)).toBeInTheDocument()
    expect(within(items[1]).getByText('Hold')).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Vehicle details' })).toHaveTextContent(
      'Transfer planned',
    )
  })

  it('says when there is no history yet', async () => {
    renderApp({ route: agingPath })

    expect(await screen.findByText('No actions have been logged yet.')).toBeInTheDocument()
  })

  it('links back to the filtered inventory it came from', async () => {
    const { user } = renderApp({ route: '/?aging=true&sort=price_asc' })

    await user.click(await screen.findByRole('link', { name: 'Toyota Corolla' }))
    await screen.findByRole('heading', { name: /Toyota Corolla/ })
    await user.click(screen.getByRole('link', { name: /Back to inventory/ }))

    await screen.findByRole('heading', { name: 'Inventory' })
    expect(screen.getByTestId('location')).toHaveTextContent('/?aging=true&sort=price_asc')
  })

  it('reports an unknown vehicle', async () => {
    renderApp({ route: '/vehicles/99999999-9999-4999-8999-999999999999' })

    expect(await screen.findByText('Vehicle not found.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Back to inventory/ })).toBeInTheDocument()
  })

  it('treats a malformed id the same way', async () => {
    server.use(
      http.get(url('/vehicles/:id'), () => apiError(400, 'validation_error', 'invalid vehicle id')),
    )
    renderApp({ route: '/vehicles/not-a-uuid' })

    expect(await screen.findByText('Vehicle not found.')).toBeInTheDocument()
  })

  it('reports a server failure', async () => {
    server.use(http.get(url('/vehicles/:id'), () => apiError(500, 'internal_error', 'boom')))
    renderApp({ route: agingPath })

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load the vehicle: boom')
  })

  it('reports a history that cannot be loaded, without hiding the vehicle', async () => {
    server.use(
      http.get(url('/vehicles/:id/actions'), () => apiError(500, 'internal_error', 'boom')),
    )
    renderApp({ route: agingPath })

    expect(await screen.findByRole('heading', { name: /Toyota Corolla/ })).toBeInTheDocument()
    expect(await screen.findByText('Could not load the history.')).toBeInTheDocument()
  })
})

describe('logging an action', () => {
  it('offers the form for an aging vehicle', async () => {
    renderApp({ route: agingPath })

    expect(await screen.findByRole('heading', { name: 'Log an action' })).toBeInTheDocument()
    const options = within(screen.getByLabelText('Action'))
      .getAllByRole('option')
      .map((o) => o.textContent)
    expect(options).toEqual([
      'Choose an action…',
      'Price reduction planned',
      'Promotion planned',
      'Transfer planned',
      'Wholesale planned',
      'Hold',
    ])
  })

  it('records the action, then shows it in the history and as the current action', async () => {
    const { posted } = statefulActions()
    const { user } = renderApp({ route: agingPath })
    await screen.findByText('No actions have been logged yet.')

    await user.selectOptions(screen.getByLabelText('Action'), 'Price reduction planned')
    await user.type(screen.getByLabelText('Note (optional)'), 'Drop by 5% on Friday')
    await user.click(screen.getByRole('button', { name: 'Log action' }))

    expect(await screen.findByText('Action logged.')).toBeInTheDocument()
    expect(posted).toEqual([
      { action_type: 'price_reduction_planned', note: 'Drop by 5% on Friday' },
    ])
    const history = await screen.findByRole('list')
    expect(within(history).getByText('Drop by 5% on Friday')).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Vehicle details' })).toHaveTextContent(
      'Price reduction planned',
    )
    // The form is ready for the next action.
    expect(screen.getByLabelText('Action')).toHaveValue('')
    expect(screen.getByLabelText('Note (optional)')).toHaveValue('')
  })

  it('sends no note when the note is blank', async () => {
    const { posted } = statefulActions()
    const { user } = renderApp({ route: agingPath })
    await screen.findByText('No actions have been logged yet.')

    await user.selectOptions(screen.getByLabelText('Action'), 'Hold')
    await user.type(screen.getByLabelText('Note (optional)'), '   ')
    await user.click(screen.getByRole('button', { name: 'Log action' }))

    await screen.findByText('Action logged.')
    expect(posted).toEqual([{ action_type: 'hold', note: null }])
  })

  it('keeps earlier actions when another is added', async () => {
    statefulActions([action({ id: 'a1', action_type: 'hold', note: 'First' })])
    const { user } = renderApp({ route: agingPath })
    await screen.findByText('First')

    await user.selectOptions(screen.getByLabelText('Action'), 'Wholesale planned')
    await user.click(screen.getByRole('button', { name: 'Log action' }))

    await waitFor(() => {
      expect(within(screen.getByRole('list')).getAllByRole('listitem')).toHaveLength(2)
    })
    expect(screen.getByText('First')).toBeInTheDocument()
  })

  it('cannot be submitted without choosing an action', async () => {
    const { posted } = statefulActions()
    const { user } = renderApp({ route: agingPath })
    await screen.findByText('No actions have been logged yet.')

    await user.click(screen.getByRole('button', { name: 'Log action' }))

    expect(posted).toEqual([])
    expect(screen.getByLabelText('Action')).toBeInvalid()
  })

  it('counts the note against its limit', async () => {
    const { user } = renderApp({ route: agingPath })
    await screen.findByText('0 / 500')

    await user.type(screen.getByLabelText('Note (optional)'), 'hello')

    expect(screen.getByText('5 / 500')).toBeInTheDocument()
    expect(screen.getByLabelText('Note (optional)')).toHaveAttribute('maxlength', '500')
  })

  it('shows what the server objected to', async () => {
    server.use(
      http.post(url('/vehicles/:id/actions'), () =>
        apiError(400, 'validation_error', 'invalid action', {
          note: 'must not contain NUL characters',
        }),
      ),
    )
    const { user } = renderApp({ route: agingPath })
    await screen.findByText('No actions have been logged yet.')

    await user.selectOptions(screen.getByLabelText('Action'), 'Hold')
    await user.click(screen.getByRole('button', { name: 'Log action' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid action')
    expect(screen.getByText('must not contain NUL characters')).toBeInTheDocument()
    expect(screen.getByLabelText('Note (optional)')).toBeInvalid()
    expect(screen.getByLabelText('Action')).toHaveValue('hold') // the input is kept for a retry
  })

  it('explains when the vehicle turns out not to be aging', async () => {
    server.use(
      http.post(url('/vehicles/:id/actions'), () =>
        apiError(409, 'vehicle_not_aging', 'only aging vehicles accept actions'),
      ),
    )
    const { user } = renderApp({ route: agingPath })
    await screen.findByText('No actions have been logged yet.')

    await user.selectOptions(screen.getByLabelText('Action'), 'Hold')
    await user.click(screen.getByRole('button', { name: 'Log action' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('not aging')
  })

  it('shows a generic message for any other failure', async () => {
    server.use(
      http.post(url('/vehicles/:id/actions'), () => new HttpResponse(null, { status: 500 })),
    )
    const { user } = renderApp({ route: agingPath })
    await screen.findByText('No actions have been logged yet.')

    await user.selectOptions(screen.getByLabelText('Action'), 'Hold')
    await user.click(screen.getByRole('button', { name: 'Log action' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Could not save the action. Please try again.',
    )
    expect(screen.getByRole('button', { name: 'Log action' })).toBeEnabled()
  })

  it('does not offer the form for stock that is not aging', async () => {
    renderApp({ route: freshPath })

    const notice = await screen.findByText(/Actions can only be logged for aging stock/)
    await waitFor(() => {
      expect(notice).toHaveTextContent('more than 90 days')
    })
    expect(notice).toHaveTextContent('in stock for 14 days')
    expect(screen.queryByRole('heading', { name: 'Log an action' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Action')).not.toBeInTheDocument()
  })

  it('still explains the rule while the threshold is unknown', async () => {
    server.use(http.get(url('/vehicles/summary'), () => apiError(500, 'internal_error', 'boom')))
    renderApp({ route: freshPath })

    const notice = await screen.findByText(/Actions can only be logged for aging stock/)
    expect(notice).not.toHaveTextContent('more than')
  })
})
