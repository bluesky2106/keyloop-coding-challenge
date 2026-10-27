import type { components } from './schema'

type Schemas = components['schemas']

export type ActionType = Schemas['ActionType']
export type Vehicle = Schemas['Vehicle']
export type VehicleAction = Schemas['VehicleAction']
export type VehiclePage = Schemas['VehiclePage']
export type InventorySummary = Schemas['InventorySummary']
export type Make = Schemas['Make']
export type Me = Schemas['Me']
export type VehicleSort = 'age_desc' | 'age_asc' | 'price_desc' | 'price_asc'
