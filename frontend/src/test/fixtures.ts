import type { InventorySummary, Make, Me, Vehicle, VehicleAction } from '../api/types'

export const ids = {
  agingVehicle: '11111111-1111-4111-8111-111111111111',
  freshVehicle: '22222222-2222-4222-8222-222222222222',
  toyota: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  corolla: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',
  camry: 'bbbbbbbb-bbbb-4bbb-8bbb-cccccccccccc',
  honda: 'aaaaaaaa-aaaa-4aaa-8aaa-bbbbbbbbbbbb',
  civic: 'bbbbbbbb-bbbb-4bbb-8bbb-dddddddddddd',
}

export const me: Me = {
  user: { id: 'u1', username: 'alice' },
  dealership: { id: 'd1', name: 'Northside Motors' },
}

export const makes: Make[] = [
  {
    id: ids.honda,
    name: 'Honda',
    models: [{ id: ids.civic, name: 'Civic' }],
  },
  {
    id: ids.toyota,
    name: 'Toyota',
    models: [
      { id: ids.corolla, name: 'Corolla' },
      { id: ids.camry, name: 'Camry' },
    ],
  },
]

export function action(over: Partial<VehicleAction> = {}): VehicleAction {
  return {
    id: 'a1',
    vehicle_id: ids.agingVehicle,
    action_type: 'price_reduction_planned',
    note: 'Drop by 5%',
    created_by: 'alice',
    created_at: '2026-03-10T12:00:00Z',
    ...over,
  }
}

export const agingVehicle: Vehicle = {
  id: ids.agingVehicle,
  vin: '1HGCM82633A004352',
  make: { id: ids.toyota, name: 'Toyota' },
  model: { id: ids.corolla, name: 'Corolla' },
  year: 2023,
  price: '24500.00',
  stocked_at: '2026-01-15',
  age_days: 120,
  is_aging: true,
  current_action: null,
}

export const freshVehicle: Vehicle = {
  id: ids.freshVehicle,
  vin: '2HGCM82633A004353',
  make: { id: ids.honda, name: 'Honda' },
  model: { id: ids.civic, name: 'Civic' },
  year: 2024,
  price: '28000.00',
  stocked_at: '2026-04-20',
  age_days: 14,
  is_aging: false,
  current_action: null,
}

export const summary: InventorySummary = { total: 50, aging: 14, aging_threshold_days: 90 }
