import { environment } from '../../../environments/environment';

/**
 * Utility helper functions for Asia/Jakarta (WIB) timezone date & time operations.
 * Enforces global timezone setting across frontend per AGENTS.md §7 & DESIGN.md §8.
 */

/**
 * Formats time as HH:mm in Asia/Jakarta timezone.
 */
export function getJakartaTimeString(
  date: Date | string | number = new Date()
): string {
  return new Intl.DateTimeFormat('en-GB', {
    timeZone: environment.timezone,
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(date));
}

/**
 * Formats full Indonesian date (e.g. "13 Agustus 2026") in Asia/Jakarta timezone.
 */
export function formatJakartaDate(
  date: Date | string | number = new Date()
): string {
  return new Intl.DateTimeFormat('id-ID', {
    timeZone: environment.timezone,
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(new Date(date));
}

/**
 * Formats full Indonesian date with weekday (e.g. "Rabu, 14 Agustus 2026") in Asia/Jakarta timezone.
 */
export function formatJakartaDayDate(
  date: Date | string | number = new Date()
): string {
  return new Intl.DateTimeFormat('id-ID', {
    timeZone: environment.timezone,
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(new Date(date));
}

/**
 * Formats date as ISO YYYY-MM-DD in Asia/Jakarta timezone.
 */
export function getJakartaISODate(
  date: Date | string | number = new Date()
): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: environment.timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date(date));
}

/**
 * Returns yesterday's date as ISO YYYY-MM-DD in Asia/Jakarta timezone.
 */
export function getJakartaYesterdayISODate(
  date: Date | string | number = new Date()
): string {
  const d = new Date(date);
  d.setDate(d.getDate() - 1);
  return getJakartaISODate(d);
}

