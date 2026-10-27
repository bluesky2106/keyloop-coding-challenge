import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router'
import { parseParams, serializeParams, type InventoryParams } from './params'

export function useInventoryParams() {
  const [search, setSearch] = useSearchParams()
  const params = useMemo(() => parseParams(search), [search])

  /**
   * Applies a change to the filters. Changing anything except the page returns to page 1,
   * since the old page number may not exist in the new result set.
   */
  const update = useCallback(
    (patch: Partial<InventoryParams>) => {
      const next = { ...params, ...patch }
      if (!('page' in patch)) next.page = 1
      setSearch(serializeParams(next))
    },
    [params, setSearch],
  )

  return { params, update }
}
