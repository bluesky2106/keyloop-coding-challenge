import { describe, expect, it } from 'vitest'
import { parseParams, serializeParams, toQuery, type InventoryParams } from './params'

const defaults: InventoryParams = {
  makeId: '',
  modelId: '',
  aging: 'all',
  minAge: '',
  maxAge: '',
  sort: 'age_desc',
  page: 1,
}

describe('parseParams', () => {
  it('returns the defaults for an empty query string', () => {
    expect(parseParams(new URLSearchParams())).toEqual(defaults)
  })

  it('reads every filter', () => {
    const search = new URLSearchParams(
      'make_id=m&model_id=o&aging=true&min_age_days=10&max_age_days=200&sort=price_asc&page=3',
    )
    expect(parseParams(search)).toEqual({
      makeId: 'm',
      modelId: 'o',
      aging: 'aging',
      minAge: '10',
      maxAge: '200',
      sort: 'price_asc',
      page: 3,
    })
  })

  it('ignores a model when no make is chosen, since the UI cannot show or change one', () => {
    expect(parseParams(new URLSearchParams('model_id=o')).modelId).toBe('')
    expect(parseParams(new URLSearchParams('make_id=m&model_id=o')).modelId).toBe('o')
  })

  it('maps aging=false to not_aging', () => {
    expect(parseParams(new URLSearchParams('aging=false')).aging).toBe('not_aging')
  })

  it.each([
    ['aging=maybe', 'aging', 'all'],
    ['sort=colour', 'sort', 'age_desc'],
    ['min_age_days=-3', 'minAge', ''],
    ['max_age_days=abc', 'maxAge', ''],
    ['page=0', 'page', 1],
    ['page=-2', 'page', 1],
    ['page=1.5', 'page', 1],
    ['page=abc', 'page', 1],
    ['page=2147483648', 'page', 1],
    ['min_age_days=100001', 'minAge', ''],
    ['max_age_days=99999999999999999999', 'maxAge', ''],
  ])('ignores a malformed value (%s)', (query, key, expected) => {
    expect(parseParams(new URLSearchParams(query))[key as keyof InventoryParams]).toBe(expected)
  })
})

describe('serializeParams', () => {
  it('leaves out defaults so URLs stay short', () => {
    expect(serializeParams(defaults).toString()).toBe('')
  })

  it('round-trips every filter', () => {
    const full: InventoryParams = {
      makeId: 'm',
      modelId: 'o',
      aging: 'not_aging',
      minAge: '5',
      maxAge: '95',
      sort: 'price_desc',
      page: 4,
    }
    expect(parseParams(serializeParams(full))).toEqual(full)
  })
})

describe('toQuery', () => {
  it('omits filters that are not set', () => {
    expect(toQuery(defaults)).toEqual({
      make_id: undefined,
      model_id: undefined,
      aging: undefined,
      min_age_days: undefined,
      max_age_days: undefined,
      sort: 'age_desc',
      page: 1,
      page_size: 25,
    })
  })

  it('converts the filters to API types', () => {
    const query = toQuery({
      ...defaults,
      makeId: 'm',
      aging: 'aging',
      minAge: '0',
      maxAge: '90',
      page: 2,
    })
    expect(query).toMatchObject({
      make_id: 'm',
      aging: true,
      min_age_days: 0,
      max_age_days: 90,
      page: 2,
    })
    expect(toQuery({ ...defaults, aging: 'not_aging' }).aging).toBe(false)
  })
})
