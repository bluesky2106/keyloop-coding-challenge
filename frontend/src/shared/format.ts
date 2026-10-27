// The API does not say which currency prices are in. USD is an assumption (A14 in
// docs/system-design.md); change it here to change every price on screen.
const CURRENCY = 'USD'

const price = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: CURRENCY,
  maximumFractionDigits: 0,
})

/** Formats a decimal string such as "24000.00" as currency. */
export function formatPrice(value: string): string {
  return price.format(Number(value))
}

const day = new Intl.DateTimeFormat('en-GB', { dateStyle: 'medium', timeZone: 'UTC' })

/** Formats a calendar date ("2026-01-31") without shifting it across time zones. */
export function formatDate(value: string): string {
  return day.format(new Date(`${value}T00:00:00Z`))
}

const moment = new Intl.DateTimeFormat('en-GB', { dateStyle: 'medium', timeStyle: 'short' })

/** Formats an instant in the viewer's time zone. */
export function formatDateTime(value: string): string {
  return moment.format(new Date(value))
}

export function plural(count: number, singular: string, pluralForm = `${singular}s`): string {
  return `${String(count)} ${count === 1 ? singular : pluralForm}`
}
