import { Link, useLocation } from 'react-router'
import type { Vehicle } from '../api/types'
import { AgingBadge } from '../shared/AgingBadge'
import { formatPrice, plural } from '../shared/format'
import { ACTION_LABELS } from '../shared/labels'

function ActionCell({ vehicle }: { vehicle: Vehicle }) {
  if (vehicle.current_action) return <>{ACTION_LABELS[vehicle.current_action.action_type]}</>
  return vehicle.is_aging ? <span className="muted">No action logged</span> : <>—</>
}

export function VehicleTable({ vehicles }: { vehicles: Vehicle[] }) {
  // The detail page uses this to link back to the same filtered view.
  const { search } = useLocation()

  return (
    <div className="table-wrap">
      <table>
        <caption className="sr-only">Vehicles in stock</caption>
        <thead>
          <tr>
            <th scope="col">Vehicle</th>
            <th scope="col">VIN</th>
            <th scope="col" className="num">
              Price
            </th>
            <th scope="col">Time in stock</th>
            <th scope="col">Current action</th>
          </tr>
        </thead>
        <tbody>
          {vehicles.map((v) => (
            <tr key={v.id} className={v.is_aging ? 'row--aging' : undefined}>
              <th scope="row">
                <Link to={`/vehicles/${v.id}`} state={{ search }}>
                  {v.make.name} {v.model.name}
                </Link>{' '}
                <span className="muted">{v.year}</span>
              </th>
              <td className="mono">{v.vin}</td>
              <td className="num">{formatPrice(v.price)}</td>
              <td>
                {plural(v.age_days, 'day')} {v.is_aging && <AgingBadge />}
              </td>
              <td>
                <ActionCell vehicle={v} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
