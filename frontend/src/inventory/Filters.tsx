import type { Make, VehicleSort } from '../api/types'
import { AgeRange } from './AgeRange'
import { SORTS, type AgingFilter, type InventoryParams } from './params'

interface Props {
  params: InventoryParams
  makes: Make[]
  onChange: (patch: Partial<InventoryParams>) => void
}

export function Filters({ params, makes, onChange }: Props) {
  const models = makes.find((m) => m.id === params.makeId)?.models ?? []
  const filtered = Boolean(
    params.makeId || params.modelId || params.aging !== 'all' || params.minAge || params.maxAge,
  )

  return (
    <section aria-label="Filters" className="card filters">
      <div className="filters__head">
        <span className="filters__title">Filters</span>
        {filtered && (
          <button
            type="button"
            className="ghost"
            onClick={() => {
              onChange({ makeId: '', modelId: '', aging: 'all', minAge: '', maxAge: '' })
            }}
          >
            Clear filters
          </button>
        )}
      </div>

      <label>
        Make
        <select
          value={params.makeId}
          onChange={(e) => {
            // A model belongs to one make, so changing the make clears the model.
            onChange({ makeId: e.target.value, modelId: '' })
          }}
        >
          <option value="">All makes</option>
          {makes.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
      </label>

      <label>
        Model
        <select
          value={params.modelId}
          disabled={!params.makeId}
          onChange={(e) => {
            onChange({ modelId: e.target.value })
          }}
        >
          <option value="">{params.makeId ? 'All models' : 'Choose a make first'}</option>
          {models.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
      </label>

      <label>
        Stock status
        <select
          value={params.aging}
          onChange={(e) => {
            onChange({ aging: e.target.value as AgingFilter })
          }}
        >
          <option value="all">All stock</option>
          <option value="aging">Aging stock</option>
          <option value="not_aging">Not aging</option>
        </select>
      </label>

      <AgeRange
        minAge={params.minAge}
        maxAge={params.maxAge}
        onApply={(minAge, maxAge) => {
          onChange({ minAge, maxAge })
        }}
      />

      <label>
        Sort by
        <select
          value={params.sort}
          onChange={(e) => {
            onChange({ sort: e.target.value as VehicleSort })
          }}
        >
          {SORTS.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
      </label>
    </section>
  )
}
