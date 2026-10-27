import { screen, waitFor, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { action, agingVehicle, freshVehicle, ids, makes } from '../test/fixtures'
import { renderApp } from '../test/render'
import { apiError, captureVehicleRequests, server, url } from '../test/server'

const lastRequest = (requests: URLSearchParams[]) => requests[requests.length - 1]

describe('the inventory list', () => {
  it('shows each vehicle with its price, time in stock and current action', async () => {
    server.use(
      http.get(url('/vehicles'), () =>
        HttpResponse.json({
          items: [
            { ...agingVehicle, current_action: action({ action_type: 'hold' }) },
            freshVehicle,
          ],
          page: 1,
          page_size: 25,
          total: 2,
        }),
      ),
    )
    renderApp()

    const rows = await screen.findAllByRole('row')
    const aging = within(rows[1])
    expect(aging.getByRole('link', { name: 'Toyota Corolla' })).toBeInTheDocument()
    expect(aging.getByText('1HGCM82633A004352')).toBeInTheDocument()
    expect(aging.getByText('$24,500')).toBeInTheDocument()
    expect(aging.getByText(/120 days/)).toBeInTheDocument()
    expect(aging.getByText('Hold')).toBeInTheDocument()
    expect(within(rows[2]).getByText(/14 days/)).toBeInTheDocument()
  })

  it('flags aging stock in words, not only colour', async () => {
    renderApp()

    const rows = await screen.findAllByRole('row')
    expect(within(rows[1]).getByText(/Aging/)).toBeInTheDocument()
    expect(within(rows[2]).queryByText(/Aging/)).not.toBeInTheDocument()
  })

  it('prompts for an action on aging vehicles that have none', async () => {
    renderApp()

    const rows = await screen.findAllByRole('row')
    expect(within(rows[1]).getByText('No action logged')).toBeInTheDocument()
    expect(within(rows[2]).getByText('—')).toBeInTheDocument()
  })

  it('requests the oldest stock first, 25 to a page', async () => {
    const requests = captureVehicleRequests()
    renderApp()
    await screen.findByText('Toyota Corolla')

    expect(Object.fromEntries(requests[0])).toEqual({
      sort: 'age_desc',
      page: '1',
      page_size: '25',
    })
  })

  it('links each vehicle to its page', async () => {
    renderApp()

    expect(await screen.findByRole('link', { name: 'Honda Civic' })).toHaveAttribute(
      'href',
      `/vehicles/${ids.freshVehicle}`,
    )
  })

  it('shows a loading state, then the vehicles', async () => {
    renderApp()

    expect(screen.getByText('Loading vehicles…')).toBeInTheDocument()
    expect(await screen.findByText('Toyota Corolla')).toBeInTheDocument()
  })

  it('says so when nothing matches', async () => {
    captureVehicleRequests({ items: [], total: 0 })
    renderApp({ route: '/?aging=true' })

    expect(await screen.findByText('No vehicles match these filters.')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('offers a way back from a page that does not exist', async () => {
    const requests = captureVehicleRequests({ items: [], total: 40 })
    const { user } = renderApp({ route: '/?page=9' })

    await user.click(await screen.findByRole('button', { name: 'Go to the first page' }))

    await screen.findByRole('heading', { name: 'Inventory' })
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/)
    await waitFor(() => {
      expect(lastRequest(requests).get('page')).toBe('1')
    })
  })

  it('reports a failure and lets the user retry', async () => {
    let attempts = 0
    server.use(
      http.get(url('/vehicles'), () => {
        attempts += 1
        return attempts === 1
          ? apiError(500, 'internal_error', 'internal server error')
          : HttpResponse.json({ items: [agingVehicle], page: 1, page_size: 25, total: 1 })
      }),
    )
    const { user } = renderApp()

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load vehicles')
    await user.click(screen.getByRole('button', { name: 'Try again' }))

    expect(await screen.findByText('Toyota Corolla')).toBeInTheDocument()
  })
})

describe('when the filters or lists cannot be loaded', () => {
  it('offers to clear filters the server rejects, since asking again would fail the same way', async () => {
    server.use(
      http.get(url('/vehicles'), ({ request }) =>
        new URL(request.url).searchParams.has('min_age_days')
          ? apiError(400, 'validation_error', 'invalid query parameters')
          : HttpResponse.json({ items: [agingVehicle], page: 1, page_size: 25, total: 1 }),
      ),
    )
    const { user } = renderApp({ route: '/?min_age_days=100&max_age_days=5' })

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'These filters cannot be applied: invalid query parameters',
    )
    expect(screen.queryByRole('button', { name: 'Try again' })).not.toBeInTheDocument()
    await user.click(screen.getAllByRole('button', { name: 'Clear filters' })[0])

    expect(await screen.findByText('Toyota Corolla')).toBeInTheDocument()
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/)
  })

  it('does not filter by a model the page cannot show', async () => {
    const requests = captureVehicleRequests()
    renderApp({ route: `/?model_id=${ids.corolla}` })

    await screen.findByText('Toyota Corolla')
    expect(requests[0].has('model_id')).toBe(false)
  })

  it('says when the make and model lists fail, and recovers on retry', async () => {
    let attempts = 0
    server.use(
      http.get(url('/makes'), () => {
        attempts += 1
        return attempts === 1
          ? apiError(500, 'internal_error', 'boom')
          : HttpResponse.json({ items: makes })
      }),
    )
    const { user } = renderApp()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Could not load the make and model lists.')
    await user.click(within(alert).getByRole('button', { name: 'Try again' }))

    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'Toyota' })).toBeInTheDocument()
    })
    expect(screen.queryByText('Could not load the make and model lists.')).not.toBeInTheDocument()
  })
})

describe('the aging summary', () => {
  it('puts the aging count and the threshold up front', async () => {
    renderApp()

    const summary = await screen.findByRole('region', { name: 'Stock summary' })
    expect(await within(summary).findByText('14')).toBeInTheDocument()
    expect(
      within(summary).getByText(/14 vehicles in stock for more than 90 days/),
    ).toBeInTheDocument()
    expect(within(summary).getByText('vehicles in stock')).toBeInTheDocument()
  })

  it('filters to aging stock from the summary', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()

    await user.click(await screen.findByRole('button', { name: 'Show aging stock' }))

    expect(screen.getByTestId('location')).toHaveTextContent('aging=true')
    await screen.findByText('Toyota Corolla')
    await waitFor(() => {
      expect(lastRequest(requests).get('aging')).toBe('true')
    })
    expect(screen.queryByRole('button', { name: 'Show aging stock' })).not.toBeInTheDocument()
  })

  it('says nothing about aging when there is no aging stock', async () => {
    server.use(
      http.get(url('/vehicles/summary'), () =>
        HttpResponse.json({ total: 5, aging: 0, aging_threshold_days: 90 }),
      ),
    )
    renderApp()

    expect(await screen.findByText(/0 vehicles in stock for more than 90 days/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Show aging stock' })).not.toBeInTheDocument()
  })

  it('uses the singular for one vehicle', async () => {
    server.use(
      http.get(url('/vehicles/summary'), () =>
        HttpResponse.json({ total: 5, aging: 1, aging_threshold_days: 60 }),
      ),
    )
    renderApp()

    expect(await screen.findByText(/1 vehicle in stock for more than 60 days/)).toBeInTheDocument()
  })

  it('reports when the summary cannot be loaded, without hiding the list', async () => {
    server.use(http.get(url('/vehicles/summary'), () => apiError(500, 'internal_error', 'boom')))
    renderApp()

    expect(await screen.findByText('Could not load the stock summary.')).toBeInTheDocument()
    expect(await screen.findByText('Toyota Corolla')).toBeInTheDocument()
  })
})

describe('filtering', () => {
  it('filters by stock status', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()
    await screen.findByText('Toyota Corolla')

    await user.selectOptions(screen.getByLabelText('Stock status'), 'Not aging')

    expect(screen.getByTestId('location')).toHaveTextContent('aging=false')
    await screen.findByText('Toyota Corolla')
    await waitFor(() => {
      expect(lastRequest(requests).get('aging')).toBe('false')
    })
  })

  it('filters by make, then lets the user narrow to a model', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()
    await screen.findByText('Toyota Corolla')
    expect(screen.getByLabelText('Model')).toBeDisabled()
    expect(screen.getByRole('option', { name: 'Choose a make first' })).toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Make'), 'Toyota')
    expect(screen.getByLabelText('Model')).toBeEnabled()
    expect(
      within(screen.getByLabelText('Model'))
        .getAllByRole('option')
        .map((o) => o.textContent),
    ).toEqual(['All models', 'Corolla', 'Camry'])
    await user.selectOptions(screen.getByLabelText('Model'), 'Camry')

    await screen.findByText('Toyota Corolla')
    await waitFor(() => {
      expect(lastRequest(requests).get('make_id')).toBe(ids.toyota)
    })
    await waitFor(() => {
      expect(lastRequest(requests).get('model_id')).toBe(ids.camry)
    })
  })

  it('clears the model when the make changes', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp({ route: `/?make_id=${ids.toyota}&model_id=${ids.camry}` })
    await screen.findByText('Toyota Corolla')

    await user.selectOptions(screen.getByLabelText('Make'), 'Honda')

    await screen.findByText('Toyota Corolla')
    await waitFor(() => {
      expect(lastRequest(requests).get('make_id')).toBe(ids.honda)
    })
    await waitFor(() => {
      expect(lastRequest(requests).has('model_id')).toBe(false)
    })
    expect(screen.getByLabelText('Model')).toHaveValue('')
  })

  it('applies an age range when the field loses focus', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()
    await screen.findByText('Toyota Corolla')

    await user.type(screen.getByLabelText('From'), '30')
    expect(requests).toHaveLength(1) // typing alone does not search
    await user.type(screen.getByLabelText('To'), '90')
    await user.tab()

    await screen.findByText('Toyota Corolla')
    await waitFor(() => {
      expect(lastRequest(requests).get('min_age_days')).toBe('30')
    })
    await waitFor(() => {
      expect(lastRequest(requests).get('max_age_days')).toBe('90')
    })
    expect(screen.getByTestId('location')).toHaveTextContent('min_age_days=30&max_age_days=90')
  })

  it('applies an age range with the Apply button', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()
    await screen.findByText('Toyota Corolla')

    await user.type(screen.getByLabelText('To'), '60')
    await user.click(screen.getByRole('button', { name: 'Apply' }))

    await waitFor(() => {
      expect(lastRequest(requests).get('max_age_days')).toBe('60')
    })
  })

  it('applies an age range when Enter is pressed', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()
    await screen.findByText('Toyota Corolla')

    await user.type(screen.getByLabelText('From'), '10{Enter}')

    await screen.findByText('Toyota Corolla')
    await waitFor(() => {
      expect(lastRequest(requests).get('min_age_days')).toBe('10')
    })
  })

  it('rejects a minimum above the maximum without searching', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp({ route: '/?max_age_days=10' })
    await screen.findByText('Toyota Corolla')

    await user.type(screen.getByLabelText('From'), '50')
    await user.tab()

    expect(await screen.findByRole('alert')).toHaveTextContent(/minimum age must not be greater/)
    expect(requests).toHaveLength(1)
    expect(screen.getByTestId('location')).toHaveTextContent('/?max_age_days=10')
  })

  it('keeps focus in the next field while the previous one is applied', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()
    await screen.findByText('Toyota Corolla')

    await user.type(screen.getByLabelText('From'), '30')
    await user.tab()
    expect(screen.getByLabelText('To')).toHaveFocus()
    await user.keyboard('90')
    await user.tab()

    await waitFor(() => {
      expect(lastRequest(requests).get('max_age_days')).toBe('90')
    })
    expect(lastRequest(requests).get('min_age_days')).toBe('30')
    expect(screen.getByLabelText('To')).toHaveValue('90')
  })

  it('shows the filters from the URL after navigating back', async () => {
    captureVehicleRequests()
    const { user } = renderApp({ route: '/?min_age_days=30&max_age_days=90' })
    await screen.findByText('Toyota Corolla')

    await user.click(screen.getByRole('button', { name: 'Clear filters' }))

    await waitFor(() => {
      expect(screen.getByLabelText('From')).toHaveValue('')
    })
    expect(screen.getByLabelText('To')).toHaveValue('')
  })

  it.each([
    ['a value that is not a number', 'abc', '', /whole numbers/],
    ['a value that is too large', '', '100001', /whole numbers/],
  ])('rejects %s without searching', async (_name, from, to, message) => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()
    await screen.findByText('Toyota Corolla')

    if (from) await user.type(screen.getByLabelText('From'), from)
    if (to) await user.type(screen.getByLabelText('To'), to)
    await user.tab()

    expect(await screen.findByRole('alert')).toHaveTextContent(message)
    expect(requests).toHaveLength(1)
  })

  it('restores the filters from the URL', async () => {
    const requests = captureVehicleRequests()
    renderApp({ route: `/?make_id=${ids.toyota}&aging=true&min_age_days=95&sort=price_asc&page=2` })

    await screen.findByText('Toyota Corolla')
    expect(screen.getByLabelText('Make')).toHaveValue(ids.toyota)
    expect(screen.getByLabelText('Stock status')).toHaveValue('aging')
    expect(screen.getByLabelText('From')).toHaveValue('95')
    expect(screen.getByLabelText('Sort by')).toHaveValue('price_asc')
    expect(Object.fromEntries(requests[0])).toMatchObject({
      make_id: ids.toyota,
      aging: 'true',
      min_age_days: '95',
      sort: 'price_asc',
      page: '2',
    })
  })

  it('changes the sort order', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()
    await screen.findByText('Toyota Corolla')

    await user.selectOptions(screen.getByLabelText('Sort by'), 'Price: high to low')

    await screen.findByText('Toyota Corolla')
    await waitFor(() => {
      expect(lastRequest(requests).get('sort')).toBe('price_desc')
    })
  })

  it('goes back to page 1 whenever a filter changes', async () => {
    const requests = captureVehicleRequests({ total: 80 })
    const { user } = renderApp({ route: '/?page=3' })
    await screen.findByText('Toyota Corolla')

    await user.selectOptions(screen.getByLabelText('Stock status'), 'Aging stock')

    await screen.findByText('Toyota Corolla')
    await waitFor(() => {
      expect(lastRequest(requests).get('page')).toBe('1')
    })
    expect(screen.getByTestId('location')).not.toHaveTextContent('page=')
  })

  it('clears every filter at once, and only offers to when one is set', async () => {
    const requests = captureVehicleRequests()
    const { user } = renderApp()
    await screen.findByText('Toyota Corolla')
    expect(screen.queryByRole('button', { name: 'Clear filters' })).not.toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Stock status'), 'Aging stock')
    await user.click(await screen.findByRole('button', { name: 'Clear filters' }))

    await screen.findByText('Toyota Corolla')
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/)
    await waitFor(() => {
      expect(lastRequest(requests).has('aging')).toBe(false)
    })
  })
})

describe('paging', () => {
  it('shows where the user is in the results', async () => {
    captureVehicleRequests({ total: 60 })
    renderApp()

    expect(await screen.findByText('1–25 of 60')).toBeInTheDocument()
    expect(screen.getByText('Page 1 of 3')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
  })

  it('moves forward and back through the pages', async () => {
    const requests = captureVehicleRequests({ total: 60 })
    const { user } = renderApp()
    await screen.findByText('Page 1 of 3')

    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Page 2 of 3')).toBeInTheDocument()
    await waitFor(() => {
      expect(lastRequest(requests).get('page')).toBe('2')
    })
    expect(screen.getByText('26–50 of 60')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Page 3 of 3')).toBeInTheDocument()
    expect(screen.getByText('51–60 of 60')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()

    await user.click(screen.getByRole('button', { name: 'Previous' }))
    expect(await screen.findByText('Page 2 of 3')).toBeInTheDocument()
  })
})
