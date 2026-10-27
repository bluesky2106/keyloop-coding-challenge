import { useSummary } from '../api/hooks'
import { plural } from '../shared/format'
import type { AgingFilter } from './params'

interface Props {
  aging: AgingFilter
  onShowAging: () => void
}

/** The headline numbers, with the aging stock called out. */
export function SummaryBar({ aging, onShowAging }: Props) {
  const { data, isPending, isError } = useSummary()

  if (isPending) return <p role="status">Loading stock summary…</p>
  if (isError)
    return (
      <p role="alert" className="error">
        Could not load the stock summary.
      </p>
    )

  const hasAging = data.aging > 0
  return (
    <section aria-label="Stock summary" className="summary">
      <div className="card summary__item">
        <span className="summary__value">{data.total}</span>
        <span className="summary__label">vehicles in stock</span>
      </div>
      <div className={`card summary__item ${hasAging ? 'summary__item--aging' : ''}`}>
        <span className="summary__value">
          {hasAging && <span aria-hidden="true">⚠ </span>}
          {data.aging}
        </span>
        <span className="summary__label">
          {plural(data.aging, 'vehicle')} in stock for more than {data.aging_threshold_days} days
        </span>
        {hasAging && aging !== 'aging' && (
          <button type="button" onClick={onShowAging}>
            Show aging stock
          </button>
        )}
      </div>
    </section>
  )
}
