import { Link, useLocation, useParams } from 'react-router'
import { useActions, useSummary, useVehicle } from '../api/hooks'
import { ApiError } from '../api/errors'
import { AgingBadge } from '../shared/AgingBadge'
import { formatDate, formatPrice, plural } from '../shared/format'
import { ACTION_LABELS } from '../shared/labels'
import { ActionForm } from './ActionForm'
import { ActionHistory } from './ActionHistory'

export function VehiclePage() {
  const { vehicleId = '' } = useParams()
  const search = (useLocation().state as { search?: string } | null)?.search ?? ''
  const vehicle = useVehicle(vehicleId)
  const actions = useActions(vehicleId)
  const summary = useSummary()

  const back = (
    <Link to={{ pathname: '/', search }} className="back">
      ← Back to inventory
    </Link>
  )

  if (vehicle.isPending) return <p role="status">Loading vehicle…</p>

  if (vehicle.isError) {
    // A malformed id (400) and an unknown or foreign vehicle (404) look the same to the user.
    const missing =
      vehicle.error instanceof ApiError &&
      (vehicle.error.status === 404 || vehicle.error.status === 400)
    return (
      <>
        {back}
        <div role="alert" className="alert alert--error">
          <p>
            {missing
              ? 'Vehicle not found.'
              : `Could not load the vehicle: ${vehicle.error.message}`}
          </p>
        </div>
      </>
    )
  }

  const v = vehicle.data
  return (
    <>
      {back}
      <div className="page-head">
        <h1>
          {v.make.name} {v.model.name} <span className="muted">{v.year}</span>
        </h1>
        {v.is_aging && <AgingBadge />}
      </div>

      <div className="vehicle">
        <div className="vehicle__main">
          <section aria-label="Vehicle details" className="card vehicle__details">
            <dl className="details">
              <div>
                <dt>VIN</dt>
                <dd className="mono">{v.vin}</dd>
              </div>
              <div>
                <dt>Price</dt>
                <dd>{formatPrice(v.price)}</dd>
              </div>
              <div>
                <dt>In stock since</dt>
                <dd>{formatDate(v.stocked_at)}</dd>
              </div>
              <div>
                <dt>Time in stock</dt>
                <dd>
                  {plural(v.age_days, 'day')} {v.is_aging && <AgingBadge />}
                </dd>
              </div>
              <div>
                <dt>Current action</dt>
                <dd>
                  {v.current_action ? ACTION_LABELS[v.current_action.action_type] : 'None logged'}
                </dd>
              </div>
            </dl>
          </section>

          <section aria-label="History" className="vehicle__history">
            <h2>History</h2>
            {actions.isPending && <p role="status">Loading history…</p>}
            {actions.isError && (
              <p role="alert" className="alert alert--error">
                Could not load the history.
              </p>
            )}
            {actions.data && <ActionHistory actions={actions.data.items} />}
          </section>
        </div>

        <section aria-label="Actions" className="vehicle__side">
          {v.is_aging ? (
            <ActionForm vehicleId={v.id} />
          ) : (
            <p className="card muted">
              Actions can only be logged for aging stock
              {summary.data &&
                ` (in stock for more than ${String(summary.data.aging_threshold_days)} days)`}
              . This vehicle has been in stock for {plural(v.age_days, 'day')}.
            </p>
          )}
        </section>
      </div>
    </>
  )
}
