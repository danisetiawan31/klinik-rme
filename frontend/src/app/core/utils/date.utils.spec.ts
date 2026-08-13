import { formatJakartaDate, getJakartaTimeString } from './date.utils';

describe('date.utils', () => {
  it('should format time string in Asia/Jakarta timezone', () => {
    const timeStr = getJakartaTimeString(new Date('2026-08-13T10:30:00Z'));
    expect(timeStr).toMatch(/^\d{2}:\d{2}$/);
  });

  it('should format full date in Indonesian locale and Asia/Jakarta timezone', () => {
    const dateStr = formatJakartaDate(new Date('2026-08-13T10:30:00Z'));
    expect(dateStr).toContain('Agustus 2026');
  });
});
