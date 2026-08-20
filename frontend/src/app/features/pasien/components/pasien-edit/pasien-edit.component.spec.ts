import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, provideRouter, Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { PasienEditComponent } from './pasien-edit.component';
import { PasienService } from '../../pasien.service';
import { Pasien } from '../../pasien.types';

const mockPasienInitial: Pasien = {
  id: 42,
  nik: '1234567890123456',
  nama: 'Budi Santoso Original',
  tanggalLahir: '1990-01-01',
  jenisKelamin: 'L',
  alamat: 'Jl. Merdeka 10',
  noTelp: '08123456789',
  consent: true,
  version: 1,
  riwayatKunjunganRingkas: [],
};

const mockPasienLatest: Pasien = {
  ...mockPasienInitial,
  nama: 'Budi Santoso Server Update',
  version: 2,
};

const mockPasienService = {
  getById: vi.fn(),
  update: vi.fn(),
};

describe('PasienEditComponent', () => {
  let component: PasienEditComponent;
  let fixture: ComponentFixture<PasienEditComponent>;
  let router: Router;

  beforeEach(async () => {
    vi.clearAllMocks();
    mockPasienService.getById.mockReturnValue(of(mockPasienInitial));

    await TestBed.configureTestingModule({
      imports: [PasienEditComponent],
      providers: [
        provideRouter([]),
        { provide: PasienService, useValue: mockPasienService },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'id' ? '42' : null),
              },
            },
          },
        },
      ],
    }).compileComponents();

    router = TestBed.inject(Router);
    fixture = TestBed.createComponent(PasienEditComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('renders skeleton placeholders when isLoadingInitial is true', () => {
    component.isLoadingInitial.set(true);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const skeletons = compiled.querySelectorAll('hlm-skeleton');
    expect(skeletons.length).toBeGreaterThanOrEqual(5);
    expect(compiled.querySelector('[aria-busy="true"]')).toBeTruthy();
  });

  it('pre-fills form with existing patient data on init', () => {
    expect(mockPasienService.getById).toHaveBeenCalledWith(42);
    expect(component.form.value).toEqual({
      nik: '1234567890123456',
      nama: 'Budi Santoso Original',
      tanggalLahir: '1990-01-01',
      jenisKelamin: 'L',
      alamat: 'Jl. Merdeka 10',
      noTelp: '08123456789',
    });
    expect(component.version()).toBe(1);
  });

  it('validates NIK format using nikFormatValidator (invalid if < 16 digits)', () => {
    component.nikCtrl.setValue('12345');
    component.nikCtrl.markAsTouched();
    expect(component.nikCtrl.errors?.['nikFormat']).toBeTruthy();
    expect(component.form.invalid).toBe(true);
  });

  it('navigates to /pasien/:id on successful 200 submit', () => {
    const navigateSpy = vi.spyOn(router, 'navigate');
    mockPasienService.update.mockReturnValue(of({ ...mockPasienInitial, version: 2 }));

    component.onSubmit();

    expect(mockPasienService.update).toHaveBeenCalledWith(42, {
      nik: '1234567890123456',
      nama: 'Budi Santoso Original',
      tanggalLahir: '1990-01-01',
      jenisKelamin: 'L',
      alamat: 'Jl. Merdeka 10',
      noTelp: '08123456789',
      version: 1,
    });
    expect(navigateSpy).toHaveBeenCalledWith(['/pasien', 42], {
      state: { successMessage: 'Biodata pasien berhasil diperbarui.' },
    });
  });

  // ── 409 Optimistic Lock Hybrid UX ──────────────────────────────────────────

  describe('409 Optimistic Lock Hybrid UX', () => {
    it('shows conflict banner and PRESERVES user edits when 409 response received', () => {
      // User modifies nama to "Budi Modified By User"
      component.namaCtrl.setValue('Budi Modified By User');

      // Server returns 409 Conflict
      mockPasienService.update.mockReturnValue(
        throwError(() => ({ status: 409, error: { error: { code: 'OPTIMISTIC_LOCK_FAILED' } } }))
      );

      component.onSubmit();
      fixture.detectChanges();

      // Conflict banner is shown
      expect(component.isConflict()).toBe(true);

      // CRITICAL: User's edits are NOT reset automatically
      expect(component.namaCtrl.value).toBe('Budi Modified By User');
    });

    it('refetches latest data and resets form ONLY when "Muat versi terbaru" button is clicked', () => {
      // Set conflict state
      component.isConflict.set(true);
      component.namaCtrl.setValue('Budi Modified By User');

      // Mock latest data refetch response
      mockPasienService.getById.mockReturnValue(of(mockPasienLatest));

      // Click "Muat versi terbaru"
      component.onReloadLatest();
      fixture.detectChanges();

      // Refetch happened
      expect(mockPasienService.getById).toHaveBeenCalledWith(42);

      // Version & form are reset to latest server data
      expect(component.version()).toBe(2);
      expect(component.namaCtrl.value).toBe('Budi Santoso Server Update');

      // Conflict banner is cleared
      expect(component.isConflict()).toBe(false);
    });
  });
});
