import { describe, expect, it } from 'vitest'
import { formatDate, formatDateTime, formatPrice, plural } from './format'

describe('formatPrice', () => {
  it('formats a decimal string as whole-unit currency', () => {
    expect(formatPrice('24500.00')).toBe('$24,500')
    expect(formatPrice('0.00')).toBe('$0')
  })
})

describe('formatDate', () => {
  it('shows the calendar date as given, whatever the viewer time zone', () => {
    expect(formatDate('2026-01-15')).toBe('15 Jan 2026')
    expect(formatDate('2026-12-31')).toBe('31 Dec 2026')
  })
})

describe('formatDateTime', () => {
  it('includes the date and the time', () => {
    const text = formatDateTime('2026-03-10T12:00:00Z')
    expect(text).toMatch(/Mar 2026/)
    expect(text).toMatch(/\d{2}:\d{2}/)
  })
})

describe('plural', () => {
  it('only pluralises when the count is not one', () => {
    expect(plural(1, 'day')).toBe('1 day')
    expect(plural(0, 'day')).toBe('0 days')
    expect(plural(14, 'vehicle')).toBe('14 vehicles')
    expect(plural(2, 'person', 'people')).toBe('2 people')
  })
})
