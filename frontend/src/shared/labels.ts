import type { ActionType } from '../api/types'

/**
 * Display names for action types. Typed as a complete Record, so adding a type to the API
 * contract is a compile error here until it has a label.
 */
export const ACTION_LABELS: Record<ActionType, string> = {
  price_reduction_planned: 'Price reduction planned',
  promotion_planned: 'Promotion planned',
  transfer_planned: 'Transfer planned',
  wholesale_planned: 'Wholesale planned',
  hold: 'Hold',
}

export const ACTION_TYPES = Object.keys(ACTION_LABELS) as ActionType[]
