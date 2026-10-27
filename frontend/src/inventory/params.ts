import type { VehicleQuery } from '../api/hooks'
import type { VehicleSort } from '../api/types'

export type AgingFilter = 'all' | 'aging' | 'not_aging'

/** The inventory filters, as held in the URL so a view can be shared and survives refresh. */
export interface InventoryParams {
  makeId: string
  modelId: string
  aging: AgingFilter
  minAge: string
  maxAge: string
  sort: VehicleSort
  page: number
}

export const PAGE_SIZE = 25
export const MAX_AGE_DAYS = 100000
export const SORTS: { value: VehicleSort; label: string }[] = [
  { value: 'age_desc', label: 'Oldest first' },
  { value: 'age_asc', label: 'Newest first' },
  { value: 'price_desc', label: 'Price: high to low' },
  { value: 'price_asc', label: 'Price: low to high' },
]

const DEFAULT_SORT: VehicleSort = 'age_desc'

function digits(value: string | null): string {
  return value !== null && /^\d+$/.test(value) ? value : ''
}

/** Reads the filters from a query string, ignoring anything malformed. */
export function parseParams(search: URLSearchParams): InventoryParams {
  const aging = search.get('aging')
  const sort = search.get('sort')
  const page = Number(search.get('page'))
  return {
    makeId: search.get('make_id') ?? '',
    modelId: search.get('model_id') ?? '',
    aging: aging === 'true' ? 'aging' : aging === 'false' ? 'not_aging' : 'all',
    minAge: digits(search.get('min_age_days')),
    maxAge: digits(search.get('max_age_days')),
    sort: SORTS.some((s) => s.value === sort) ? (sort as VehicleSort) : DEFAULT_SORT,
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  }
}

/** Writes the filters to a query string, leaving out defaults so URLs stay short. */
export function serializeParams(params: InventoryParams): URLSearchParams {
  const search = new URLSearchParams()
  if (params.makeId) search.set('make_id', params.makeId)
  if (params.modelId) search.set('model_id', params.modelId)
  if (params.aging !== 'all') search.set('aging', String(params.aging === 'aging'))
  if (params.minAge) search.set('min_age_days', params.minAge)
  if (params.maxAge) search.set('max_age_days', params.maxAge)
  if (params.sort !== DEFAULT_SORT) search.set('sort', params.sort)
  if (params.page > 1) search.set('page', String(params.page))
  return search
}

/** Converts the filters to an API request. */
export function toQuery(params: InventoryParams): VehicleQuery {
  return {
    make_id: params.makeId || undefined,
    model_id: params.modelId || undefined,
    aging: params.aging === 'all' ? undefined : params.aging === 'aging',
    min_age_days: params.minAge ? Number(params.minAge) : undefined,
    max_age_days: params.maxAge ? Number(params.maxAge) : undefined,
    sort: params.sort,
    page: params.page,
    page_size: PAGE_SIZE,
  }
}
