interface Props {
  page: number
  pageSize: number
  total: number
  onPage: (page: number) => void
}

export function Pagination({ page, pageSize, total, onPage }: Props) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  const first = total === 0 ? 0 : (page - 1) * pageSize + 1
  const last = Math.min(page * pageSize, total)

  return (
    <nav aria-label="Pagination" className="pagination">
      <span>
        {first}–{last} of {total}
      </span>
      <button
        type="button"
        className="secondary"
        disabled={page <= 1}
        onClick={() => {
          onPage(page - 1)
        }}
      >
        Previous
      </button>
      <span aria-current="page">
        Page {page} of {pages}
      </span>
      <button
        type="button"
        className="secondary"
        disabled={page >= pages}
        onClick={() => {
          onPage(page + 1)
        }}
      >
        Next
      </button>
    </nav>
  )
}
