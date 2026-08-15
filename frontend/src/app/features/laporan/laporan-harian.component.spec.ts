import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { toast } from '@spartan-ng/brain/sonner';
import { getJakartaISODate } from '../../core/utils/date.utils';
import { LaporanHarianComponent } from './laporan-harian.component';
import { LaporanService } from './laporan.service';
import { LaporanHarian } from './laporan.types';

describe('LaporanHarianComponent', () => {
  let component: LaporanHarianComponent;
  let fixture: ComponentFixture<LaporanHarianComponent>;
  let mockLaporanService: { getLaporanHarian: ReturnType<typeof vi.fn> };

  const mockData: LaporanHarian = {
    tanggal: '2026-08-14',
    totalKunjungan: 25,
    totalSelesai: 22,
    totalTidakHadir: 3,
  };

  beforeEach(async () => {
    mockLaporanService = {
      getLaporanHarian: vi.fn().mockReturnValue(of(mockData)),
    };

    await TestBed.configureTestingModule({
      imports: [LaporanHarianComponent],
      providers: [{ provide: LaporanService, useValue: mockLaporanService }],
    }).compileComponents();

    fixture = TestBed.createComponent(LaporanHarianComponent);
    component = fixture.componentInstance;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should create and initialize default tanggalFilter to today in Asia/Jakarta timezone', () => {
    const todayJakarta = getJakartaISODate();
    expect(component.tanggalFilter()).toBe(todayJakarta);

    fixture.detectChanges();
    expect(mockLaporanService.getLaporanHarian).toHaveBeenCalledWith(todayJakarta);
    expect(component.laporan()).toEqual(mockData);
  });

  it('should render 3 summary cards with accurate metrics', () => {
    fixture.detectChanges();

    const compiled = fixture.nativeElement;
    expect(compiled.textContent).toContain('Total Kunjungan');
    expect(compiled.textContent).toContain('25');
    expect(compiled.textContent).toContain('Selesai Dilayani');
    expect(compiled.textContent).toContain('22');
    expect(compiled.textContent).toContain('Tidak Hadir');
    expect(compiled.textContent).toContain('3');
  });

  it('should refetch laporan when date filter changes via onTanggalChange', () => {
    fixture.detectChanges();
    mockLaporanService.getLaporanHarian.mockClear();

    const updatedData: LaporanHarian = {
      tanggal: '2026-08-10',
      totalKunjungan: 18,
      totalSelesai: 17,
      totalTidakHadir: 1,
    };
    mockLaporanService.getLaporanHarian.mockReturnValue(of(updatedData));

    component.onTanggalChange('2026-08-10');
    fixture.detectChanges();

    expect(component.tanggalFilter()).toBe('2026-08-10');
    expect(mockLaporanService.getLaporanHarian).toHaveBeenCalledWith('2026-08-10');
    expect(component.laporan()).toEqual(updatedData);

    const compiled = fixture.nativeElement;
    expect(compiled.textContent).toContain('18');
    expect(compiled.textContent).toContain('17');
    expect(compiled.textContent).toContain('1');
  });

  it('should display error toast when getLaporanHarian fails with 400 TANGGAL_INVALID', () => {
    const toastSpy = vi.spyOn(toast, 'error').mockImplementation(() => '' as any);
    mockLaporanService.getLaporanHarian.mockReturnValue(
      throwError(() => ({
        error: {
          code: 'TANGGAL_INVALID',
          message: 'Format tanggal tidak valid. Gunakan format YYYY-MM-DD.',
        },
      }))
    );

    fixture.detectChanges();

    expect(component.errorMessage()).toBe(
      'Format tanggal tidak valid. Gunakan format YYYY-MM-DD.'
    );
    expect(toastSpy).toHaveBeenCalledWith(
      'Format tanggal tidak valid. Gunakan format YYYY-MM-DD.'
    );
  });
});
