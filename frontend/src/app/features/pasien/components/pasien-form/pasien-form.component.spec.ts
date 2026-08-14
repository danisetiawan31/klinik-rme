import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { PasienFormComponent } from './pasien-form.component';
import { PasienService } from '../../pasien.service';

const mockPasienService = {
  searchByNik: vi.fn(),
  create: vi.fn(),
};

describe('PasienFormComponent', () => {
  let component: PasienFormComponent;
  let fixture: ComponentFixture<PasienFormComponent>;

  beforeEach(async () => {
    vi.clearAllMocks();

    await TestBed.configureTestingModule({
      imports: [PasienFormComponent],
      providers: [
        provideRouter([]),
        { provide: PasienService, useValue: mockPasienService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(PasienFormComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  // ── Form validation: consent ──────────────────────────────────────────────

  describe('form validation — consent', () => {
    it('form is invalid when consent is false', () => {
      fillRequiredFields(component);
      component.consentCtrl.setValue(false);
      expect(component.form.invalid).toBe(true);
      expect(component.consentCtrl.errors?.['required']).toBeTruthy();
    });

    it('form is valid when consent is true and required fields are filled', () => {
      fillRequiredFields(component);
      component.consentCtrl.setValue(true);
      expect(component.consentCtrl.valid).toBe(true);
    });

    it('submit is blocked and markAllAsTouched is called when consent unchecked', () => {
      fillRequiredFields(component);
      component.consentCtrl.setValue(false);

      const markSpy = vi.spyOn(component.form, 'markAllAsTouched');
      component.onSubmit();

      expect(markSpy).toHaveBeenCalled();
      expect(mockPasienService.create).not.toHaveBeenCalled();
    });
  });

  // ── Form validation: NIK format ───────────────────────────────────────────

  describe('form validation — NIK format', () => {
    it('NIK empty is valid (nullable)', () => {
      component.nikCtrl.setValue('');
      component.nikCtrl.markAsTouched();
      expect(component.nikCtrl.errors?.['nikFormat']).toBeFalsy();
      expect(component.nikCtrl.valid).toBe(true);
    });

    it('NIK with fewer than 16 digits is invalid', () => {
      component.nikCtrl.setValue('123456789012345');
      component.nikCtrl.markAsTouched();
      expect(component.nikCtrl.errors?.['nikFormat']).toBeTruthy();
    });

    it('NIK with more than 16 digits is invalid', () => {
      component.nikCtrl.setValue('12345678901234567');
      component.nikCtrl.markAsTouched();
      expect(component.nikCtrl.errors?.['nikFormat']).toBeTruthy();
    });

    it('NIK with 16 non-numeric chars is invalid', () => {
      component.nikCtrl.setValue('1234567890123abc');
      component.nikCtrl.markAsTouched();
      expect(component.nikCtrl.errors?.['nikFormat']).toBeTruthy();
    });

    it('NIK with exactly 16 digits is valid', () => {
      component.nikCtrl.setValue('1234567890123456');
      component.nikCtrl.markAsTouched();
      expect(component.nikCtrl.errors?.['nikFormat']).toBeFalsy();
      expect(component.nikCtrl.valid).toBe(true);
    });
  });

  // ── NIK duplicate check trigger ───────────────────────────────────────────

  describe('NIK duplicate check', () => {
    it('does NOT call searchByNik when NIK is less than 16 digits', () => {
      component.nikCtrl.setValue('123456789012345'); // 15 digits
      component.onNikInput();
      expect(mockPasienService.searchByNik).not.toHaveBeenCalled();
    });

    it('does NOT call searchByNik when NIK is empty', () => {
      component.nikCtrl.setValue('');
      component.onNikInput();
      expect(mockPasienService.searchByNik).not.toHaveBeenCalled();
    });

    it('does NOT call searchByNik when NIK contains non-numeric chars', () => {
      component.nikCtrl.setValue('123456789012345a');
      component.onNikInput();
      expect(mockPasienService.searchByNik).not.toHaveBeenCalled();
    });

    it('calls searchByNik exactly when NIK is exactly 16 digits', () => {
      mockPasienService.searchByNik.mockReturnValue(of([]));
      component.nikCtrl.setValue('1234567890123456');
      component.onNikInput();
      expect(mockPasienService.searchByNik).toHaveBeenCalledOnce();
      expect(mockPasienService.searchByNik).toHaveBeenCalledWith('1234567890123456');
    });

    it('sets nikDuplicateWarning when a match is found', () => {
      const match = { id: 1, nik: '1234567890123456', nama: 'Budi Santoso', tanggalLahir: '1990-01-01' };
      mockPasienService.searchByNik.mockReturnValue(of([match]));
      component.nikCtrl.setValue('1234567890123456');
      component.onNikInput();
      expect(component.nikDuplicateWarning()).toEqual(match);
    });

    it('warning is non-blocking: submit button remains active when warning is shown', () => {
      const match = { id: 1, nik: '1234567890123456', nama: 'Budi Santoso', tanggalLahir: '1990-01-01' };
      mockPasienService.searchByNik.mockReturnValue(of([match]));
      component.nikCtrl.setValue('1234567890123456');
      component.onNikInput();

      // Warning is shown
      expect(component.nikDuplicateWarning()).toEqual(match);

      // Form is not disabled — staff can still submit
      expect(component.form.disabled).toBe(false);
      expect(component.isLoading()).toBe(false);
    });

    it('clears nikDuplicateWarning when NIK changes to non-16-digit value', () => {
      component.nikDuplicateWarning.set({ id: 1, nik: '1234567890123456', nama: 'Budi', tanggalLahir: '1990-01-01' });
      component.nikCtrl.setValue('12345');
      component.onNikInput();
      expect(component.nikDuplicateWarning()).toBeNull();
    });

    it('silently ignores searchByNik error — no errorMessage set, isCheckingNik reset', () => {
      mockPasienService.searchByNik.mockReturnValue(throwError(() => new Error('network')));
      component.nikCtrl.setValue('1234567890123456');
      component.onNikInput();
      expect(component.errorMessage()).toBeNull();
      expect(component.isCheckingNik()).toBe(false);
    });
  });
});

// ── helpers ─────────────────────────────────────────────────────────────────

function fillRequiredFields(component: PasienFormComponent): void {
  component.namaCtrl.setValue('Budi Santoso');
  component.tanggalLahirCtrl.setValue('1990-01-01');
  component.jenisKelaminCtrl.setValue('L');
  component.alamatCtrl.setValue('Jl. Merdeka No. 1');
  component.noTelpCtrl.setValue('08123456789');
}
