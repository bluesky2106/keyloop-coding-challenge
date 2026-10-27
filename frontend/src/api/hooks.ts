import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './client'
import { toApiError } from './errors'
import type { ActionType, VehicleSort } from './types'

/** Unwraps an openapi-fetch result: the data on success, a thrown ApiError otherwise. */
async function unwrap<T>(
  result: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<T> {
  const { data, error, response } = await result
  if (data === undefined) throw toApiError(response.status, error)
  return data
}

export function useMe() {
  return useQuery({ queryKey: ['me'], queryFn: () => unwrap(api.GET('/me')), staleTime: Infinity })
}

export function useMakes() {
  return useQuery({
    queryKey: ['makes'],
    queryFn: () => unwrap(api.GET('/makes')),
    staleTime: Infinity,
  })
}

export function useSummary() {
  return useQuery({
    queryKey: ['summary'],
    queryFn: () => unwrap(api.GET('/vehicles/summary')),
  })
}

export interface VehicleQuery {
  make_id?: string
  model_id?: string
  aging?: boolean
  min_age_days?: number
  max_age_days?: number
  sort: VehicleSort
  page: number
  page_size: number
}

export function useVehicles(query: VehicleQuery) {
  return useQuery({
    queryKey: ['vehicles', query],
    queryFn: () => unwrap(api.GET('/vehicles', { params: { query } })),
    // Keep showing the current page while the next one loads, so filters do not flash empty.
    placeholderData: keepPreviousData,
  })
}

export function useVehicle(vehicleId: string) {
  return useQuery({
    queryKey: ['vehicle', vehicleId],
    queryFn: () => unwrap(api.GET('/vehicles/{vehicleId}', { params: { path: { vehicleId } } })),
  })
}

export function useActions(vehicleId: string) {
  return useQuery({
    queryKey: ['actions', vehicleId],
    queryFn: () =>
      unwrap(api.GET('/vehicles/{vehicleId}/actions', { params: { path: { vehicleId } } })),
  })
}

export interface LogActionInput {
  action_type: ActionType
  note: string | null
}

export function useLogAction(vehicleId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: LogActionInput) =>
      unwrap(
        api.POST('/vehicles/{vehicleId}/actions', {
          params: { path: { vehicleId } },
          body: input,
        }),
      ),
    onSuccess: async () => {
      // The history, the vehicle's current action and the list all change.
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['actions', vehicleId] }),
        queryClient.invalidateQueries({ queryKey: ['vehicle', vehicleId] }),
        queryClient.invalidateQueries({ queryKey: ['vehicles'] }),
      ])
    },
  })
}
