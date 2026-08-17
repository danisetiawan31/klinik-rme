import {
  formatJakartaDate,
  formatJakartaDayDate,
  getJakartaISODate,
  getJakartaTimeString,
} from './date.utils';

describe('date.utils', () => {
  it('should format time string in Asia/Jakarta timezone', () => {
    const timeStr = getJakartaTimeString(new Date('2026-08-13T10:30:00Z'));
    expect(timeStr).toMatch(/^\d{2}:\d{2}$/);
  });

  it('should format full date in Indonesian locale and Asia/Jakarta timezone', () => {
    const dateStr = formatJakartaDate(new Date('2026-08-13T10:30:00Z'));
    expect(dateStr).toContain('Agustus 2026');
  });

  it('should format date with weekday in Indonesian locale', () => {
    const dayDateStr = formatJakartaDayDate(new Date('2026-08-13T10:30:00Z'));
    expect(dayDateStr).toContain('Agustus 2026');
    expect(dayDateStr).toMatch(/^(Senin|Selasa|Rabu|Kamis|Jumat|Sabtu|Minggu)/);
  });

  it('should format ISO YYYY-MM-DD date in Asia/Jakarta timezone', () => {
    // 2026-08-13T23:00:00Z is 2026-08-14T06:00:00+07:00 in Asia/Jakarta
    const isoDateStr = getJakartaISODate(new Date('2026-08-13T23:00:00Z'));
    expect(isoDateStr).toBe('2026-08-14');
  });
});

