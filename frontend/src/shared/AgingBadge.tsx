/** Marks aging stock with an icon and text, so the signal never depends on colour alone. */
export function AgingBadge() {
  return (
    <span className="badge badge--aging">
      <span aria-hidden="true">⚠</span> Aging
    </span>
  )
}
