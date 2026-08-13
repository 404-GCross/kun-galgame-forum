import { describe, it, expect } from 'vitest'
import { isValidURL, isValidName } from './validate'

describe('isValidURL', () => {
  it('accepts valid http/https/etc URLs', () => {
    expect(isValidURL('https://example.com')).toBe(true)
    expect(isValidURL('http://localhost:1007/foo?q=1')).toBe(true)
    expect(isValidURL('ftp://files.example.com/path')).toBe(true)
  })
  it('rejects junk strings', () => {
    expect(isValidURL('not-a-url')).toBe(false)
    expect(isValidURL('example.com')).toBe(false)
    expect(isValidURL('')).toBe(false)
  })
})

describe('isValidName', () => {
  it('accepts normal Latin / CJK names without whitespace or invisibles', () => {
    expect(isValidName('KUN')).toBe(true)
    expect(isValidName('Yuki_Onna1007')).toBe(true)
    expect(isValidName('小恋')).toBe(true)
  })

  it('rejects names containing ASCII space (\\u0020 blocked by design)', () => {
    expect(isValidName('Yuki Onna')).toBe(false)
  })

  it('rejects names containing zero-width / invisible characters', () => {
    expect(isValidName('user​ name')).toBe(false)
    expect(isValidName('a‌b')).toBe(false)
    expect(isValidName('admin﻿')).toBe(false)
    expect(isValidName('foo bar')).toBe(false)
  })

  it('rejects names with control / bidi-override characters', () => {
    expect(isValidName('safe‮txt')).toBe(false)
  })
})
