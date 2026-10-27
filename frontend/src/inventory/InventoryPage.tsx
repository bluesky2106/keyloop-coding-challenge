import { useMakes, useVehicles } from '../api/hooks'
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

  return (
    <>
      <h1>Inventory</h1>
      <SummaryBar
        aging={params.aging}
        onShowAging={() => {
          update({ aging: 'aging' })
        }}
      />
      <Filters params={params} makes={makes.data?.items ?? []} onChange={update} />

      {vehicles.isPending && <p role="status">Loading vehicles…</p>}

      {vehicles.isError && (
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
      )}

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
