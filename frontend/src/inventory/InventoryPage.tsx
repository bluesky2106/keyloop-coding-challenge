import { useMakes, useVehicles } from '../api/hooks'
import { ApiError } from '../api/errors'
import { Filters } from './Filters'
import { Pagination } from './Pagination'
import { SummaryBar } from './SummaryBar'
import { VehicleTable } from './VehicleTable'
import { PAGE_SIZE, toQuery } from './params'
import { useInventoryParams } from './useInventoryParams'

export function InventoryPage() {
  const { params, update } = useInventoryParams()
  const makes = useMakes()
  const vehicles = useVehicles(toQuery(params))
  const clearFilters = () => {
    update({ makeId: '', modelId: '', aging: 'all', minAge: '', maxAge: '' })
  }

  return (
    <>
      <h1>Inventory</h1>
      <SummaryBar
        aging={params.aging}
        onShowAging={() => {
          update({ aging: 'aging' })
        }}
      />
      {makes.isError && (
        <div role="alert" className="error">
          <p>Could not load the make and model lists.</p>
          <button
            type="button"
            onClick={() => {
              void makes.refetch()
            }}
          >
            Try again
          </button>
        </div>
      )}
      <Filters params={params} makes={makes.data?.items ?? []} onChange={update} />

      {vehicles.isPending && <p role="status">Loading vehicles…</p>}

      {vehicles.isError &&
        (vehicles.error instanceof ApiError && vehicles.error.status < 500 ? (
          // The server rejected the filters themselves (for example a hand-edited URL), so
          // asking again would fail the same way.
          <div role="alert" className="error">
            <p>These filters cannot be applied: {vehicles.error.message}</p>
            <button type="button" onClick={clearFilters}>
              Clear filters
            </button>
          </div>
        ) : (
          <div role="alert" className="error">
            <p>Could not load vehicles: {vehicles.error.message}</p>
            <button
              type="button"
              onClick={() => {
                void vehicles.refetch()
              }}
            >
              Try again
            </button>
          </div>
        ))}

      {vehicles.data &&
        (vehicles.data.items.length === 0 ? (
          <div className="card empty">
            <p>No vehicles match these filters.</p>
            {params.page > 1 && (
              <button
                type="button"
                onClick={() => {
                  update({ page: 1 })
                }}
              >
                Go to the first page
              </button>
            )}
          </div>
        ) : (
          <div aria-busy={vehicles.isPlaceholderData}>
            <VehicleTable vehicles={vehicles.data.items} />
            <Pagination
              page={params.page}
              pageSize={PAGE_SIZE}
              total={vehicles.data.total}
              onPage={(page) => {
                update({ page })
              }}
            />
          </div>
        ))}
    </>
  )
}
