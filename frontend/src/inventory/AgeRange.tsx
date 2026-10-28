import { useState, type SyntheticEvent } from 'react'
import { MAX_AGE_DAYS } from './params'

interface Props {
  minAge: string
  maxAge: string
  onApply: (minAge: string, maxAge: string) => void
}

function validate(min: string, max: string): string | null {
  for (const value of [min, max]) {
    if (value !== '' && (!/^\d+$/.test(value) || Number(value) > MAX_AGE_DAYS)) {
      return `Ages must be whole numbers between 0 and ${String(MAX_AGE_DAYS)}.`
    }
  }
  if (min !== '' && max !== '' && Number(min) > Number(max)) {
    return 'The minimum age must not be greater than the maximum.'
  }
  return null
}

/**
 * Minimum and maximum age in days. The values are applied when a field loses focus or Enter
 * is pressed, so typing "9" on the way to "90" does not trigger a request.
 */
export function AgeRange({ minAge, maxAge, onApply }: Props) {
  const [min, setMin] = useState(minAge)
  const [max, setMax] = useState(maxAge)
  const [error, setError] = useState<string | null>(null)

  // The URL is the source of truth: when it changes (browser back, "Clear filters") the
  // drafts follow it. Doing this during render, not by remounting, keeps focus where it is.
  const [applied, setApplied] = useState({ minAge, maxAge })
  if (applied.minAge !== minAge || applied.maxAge !== maxAge) {
    setApplied({ minAge, maxAge })
    setMin(minAge)
    setMax(maxAge)
    setError(null)
  }

  function apply(): void {
    const problem = validate(min, max)
    setError(problem)
    if (problem === null && (min !== minAge || max !== maxAge)) onApply(min, max)
  }

  function onSubmit(event: SyntheticEvent): void {
    event.preventDefault()
    apply()
  }

  return (
    <form onSubmit={onSubmit} className="age-range">
      <fieldset>
        <legend>Age in stock (days)</legend>
        <label>
          <span className="sr-only">From</span>
          <input
            inputMode="numeric"
            placeholder="From"
            value={min}
            onChange={(e) => {
              setMin(e.target.value)
            }}
            onBlur={apply}
            aria-invalid={error !== null}
          />
        </label>
        <label>
          <span className="sr-only">To</span>
          <input
            inputMode="numeric"
            placeholder="To"
            value={max}
            onChange={(e) => {
              setMax(e.target.value)
            }}
            onBlur={apply}
            aria-invalid={error !== null}
          />
        </label>
        <button type="submit" className="secondary">
          Apply
        </button>
      </fieldset>
      {error && (
        <p role="alert" className="error age-range__error">
          {error}
        </p>
      )}
    </form>
  )
}
