import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { formatRelativeTime } from './timeUtils';

describe('formatRelativeTime', () => {
  const mockNow = new Date('2024-01-15T12:00:00.000Z');

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(mockNow);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should return "Just now" for times less than a minute ago', () => {
    const recent = new Date('2024-01-15T11:59:30.000Z'); // 30 seconds ago
    expect(formatRelativeTime(recent)).toBe('Just now');
  });

  it('should return minutes ago for times less than an hour', () => {
    const fiveMinutesAgo = new Date('2024-01-15T11:55:00.000Z');
    expect(formatRelativeTime(fiveMinutesAgo)).toBe('5m ago');

    const thirtyMinutesAgo = new Date('2024-01-15T11:30:00.000Z');
    expect(formatRelativeTime(thirtyMinutesAgo)).toBe('30m ago');
  });

  it('should return hours ago for times less than a day', () => {
    const twoHoursAgo = new Date('2024-01-15T10:00:00.000Z');
    expect(formatRelativeTime(twoHoursAgo)).toBe('2h ago');

    const tenHoursAgo = new Date('2024-01-15T02:00:00.000Z');
    expect(formatRelativeTime(tenHoursAgo)).toBe('10h ago');
  });

  it('should return "Yesterday" for exactly one day ago', () => {
    const yesterday = new Date('2024-01-14T12:00:00.000Z');
    expect(formatRelativeTime(yesterday)).toBe('Yesterday');
  });

  it('should return days ago for times less than a week', () => {
    const twoDaysAgo = new Date('2024-01-13T12:00:00.000Z');
    expect(formatRelativeTime(twoDaysAgo)).toBe('2d ago');

    const fiveDaysAgo = new Date('2024-01-10T12:00:00.000Z');
    expect(formatRelativeTime(fiveDaysAgo)).toBe('5d ago');
  });

  it('should return date format for times more than a week ago', () => {
    const eightDaysAgo = new Date('2024-01-07T12:00:00.000Z');
    expect(formatRelativeTime(eightDaysAgo)).toBe('1/7');

    const lastMonth = new Date('2023-12-15T12:00:00.000Z');
    expect(formatRelativeTime(lastMonth)).toBe('12/15');
  });

  it('should accept string timestamps', () => {
    const fiveMinutesAgo = '2024-01-15T11:55:00.000Z';
    expect(formatRelativeTime(fiveMinutesAgo)).toBe('5m ago');
  });

  it('should accept Date objects', () => {
    const twoHoursAgo = new Date('2024-01-15T10:00:00.000Z');
    expect(formatRelativeTime(twoHoursAgo)).toBe('2h ago');
  });
});
