import { getGenconDates } from './gencon-dates';

describe('GENCON_DATES Lookup Table Integrity', () => {
  it('should contain accurate date ranges for all supported years 2018-2030', () => {
    const expectedRanges: Record<number, { startDate: string; endDate: string }> = {
      2018: { startDate: '2018-08-01', endDate: '2018-08-05' },
      2019: { startDate: '2019-07-31', endDate: '2019-08-04' },
      2020: { startDate: '2020-07-29', endDate: '2020-08-02' },
      2021: { startDate: '2021-09-15', endDate: '2021-09-19' },
      2022: { startDate: '2022-08-03', endDate: '2022-08-07' },
      2023: { startDate: '2023-08-02', endDate: '2023-08-06' },
      2024: { startDate: '2024-07-31', endDate: '2024-08-04' },
      2025: { startDate: '2025-07-30', endDate: '2025-08-03' },
      2026: { startDate: '2026-07-29', endDate: '2026-08-02' },
      2027: { startDate: '2027-08-04', endDate: '2027-08-08' },
      2028: { startDate: '2028-08-02', endDate: '2028-08-06' },
      2029: { startDate: '2029-08-01', endDate: '2029-08-05' },
      2030: { startDate: '2030-07-30', endDate: '2030-08-04' }
    };

    for (const [yearStr, expected] of Object.entries(expectedRanges)) {
      const year = parseInt(yearStr, 10);
      const actual = getGenconDates(year);
      expect(actual.startDate).toBe(expected.startDate);
      expect(actual.endDate).toBe(expected.endDate);
    }
  });

  it('should return fallback range if year is outside table', () => {
    const fallback = getGenconDates(2099);
    expect(fallback.startDate).toBe('2099-07-29');
    expect(fallback.endDate).toBe('2099-08-02');
  });
});
