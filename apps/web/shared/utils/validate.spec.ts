import { describe, it, expect } from 'vitest'
import {
  isValidTimestamp,
  isValidURL,
  isValidEmail,
  isValidName
} from './validate'

describe('isValidTimestamp', () => {
  it('accepts 10-digit (epoch s) and 13-digit (epoch ms)', () => {
    expect(isValidTimestamp(1_700_000_000)).toBe(true)
    expect(isValidTimestamp(1_700_000_000_000)).toBe(true)
  })
  it('rejects other lengths', () => {
    expect(isValidTimestamp(0)).toBe(false)
    expect(isValidTimestamp(123)).toBe(false)
    expect(isValidTimestamp(99999999999999)).toBe(false)
  })
})

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

describe('isValidEmail', () => {
  it('accepts common formats', () => {
    expect(isValidEmail('user@example.com')).toBe(true)
    expect(isValidEmail('first.last+tag@sub.example.co.uk')).toBe(true)
  })
  it('rejects malformed addresses', () => {
    expect(isValidEmail('not-an-email')).toBe(false)
    expect(isValidEmail('a@b')).toBe(false)
    expect(isValidEmail('@example.com')).toBe(false)
    expect(isValidEmail('user@')).toBe(false)
    expect(isValidEmail('user @ example.com')).toBe(false)
    expect(isValidEmail('')).toBe(false)
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
