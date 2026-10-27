import type { VehicleAction } from '../api/types'
import { formatDateTime } from '../shared/format'
import { ACTION_LABELS } from '../shared/labels'

export function ActionHistory({ actions }: { actions: VehicleAction[] }) {
  if (actions.length === 0) return <p className="muted">No actions have been logged yet.</p>

  return (
    <ol className="history">
      {actions.map((a) => (
        <li key={a.id} className="card">
          <strong>{ACTION_LABELS[a.action_type]}</strong>
          {a.note && <p>{a.note}</p>}
          <p className="muted">
            {a.created_by} · <time dateTime={a.created_at}>{formatDateTime(a.created_at)}</time>
          </p>
        </li>
      ))}
    </ol>
  )
}
