import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { toast } from '@spartan-ng/brain/sonner';
import { AuthService } from '../../../core/auth/auth.service';
import { SetPasswordComponent } from './set-password.component';

describe('SetPasswordComponent', () => {
  let component: SetPasswordComponent;
  let fixture: ComponentFixture<SetPasswordComponent>;
  let authServiceSpy: {
    resetPassword: ReturnType<typeof vi.fn>;
  };

  const createComponent = async (queryParams: Record<string, string>) => {
    authServiceSpy = {
      resetPassword: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [SetPasswordComponent],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authServiceSpy },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              queryParams,
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(SetPasswordComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  };

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('With valid token in queryParams', () => {
    beforeEach(async () => {
      await createComponent({ token: 'valid-test-token-123' });
    });

    it('should initialize token from queryParams and not set token error', () => {
      expect(component.token()).toBe('valid-test-token-123');
      expect(component.hasTokenError()).toBe(false);
    });

    it('should validate password min 8 chars and confirmation mismatch', () => {
      component.resetForm.controls.passwordBaru.setValue('short');
      component.resetForm.controls.passwordBaru.markAsTouched();
      component.resetForm.controls.konfirmasiPassword.setValue('different');
      component.resetForm.controls.konfirmasiPassword.markAsTouched();
      fixture.detectChanges();

      expect(component.resetForm.controls.passwordBaru.errors?.['minlength']).toBeTruthy();
      expect(component.resetForm.errors?.['passwordsMismatch']).toBe(true);

      const compiled = fixture.nativeElement as HTMLElement;
      expect(compiled.textContent).toContain('Password baru minimal 8 karakter');
      expect(compiled.textContent).toContain('Konfirmasi password tidak cocok');
    });

    it('should call authService.resetPassword and render Success State card on valid submit', () => {
      authServiceSpy.resetPassword.mockReturnValue(of(undefined));

      component.resetForm.setValue({
        passwordBaru: 'passwordBaru123',
        konfirmasiPassword: 'passwordBaru123',
      });

      component.onSubmit();
      fixture.detectChanges();

      expect(authServiceSpy.resetPassword).toHaveBeenCalledWith('valid-test-token-123', 'passwordBaru123');
      expect(component.isSubmitted()).toBe(true);

      const compiled = fixture.nativeElement as HTMLElement;
      expect(compiled.textContent).toContain('Password Berhasil Diubah');
      expect(compiled.textContent).toContain('Masuk ke Halaman Login');
    });

    it('should render Token Error State card when authService.resetPassword returns HTTP 400 INVALID_TOKEN', () => {
      authServiceSpy.resetPassword.mockReturnValue(
        throwError(() => ({
          status: 400,
          error: {
            error: {
              code: 'INVALID_TOKEN',
              message: 'Token reset password tidak valid atau sudah kadaluarsa',
            },
          },
        }))
      );

      component.resetForm.setValue({
        passwordBaru: 'password123',
        konfirmasiPassword: 'password123',
      });

      component.onSubmit();
      fixture.detectChanges();

      expect(component.hasTokenError()).toBe(true);

      const compiled = fixture.nativeElement as HTMLElement;
      expect(compiled.textContent).toContain('Link Tidak Valid atau Kadaluarsa');
      expect(compiled.textContent).toContain('Minta Link Baru');
    });

    it('should trigger toast.error and keep form active when technical HTTP 500 error occurs', () => {
      const toastSpy = vi.spyOn(toast, 'error').mockImplementation(() => '' as any);
      authServiceSpy.resetPassword.mockReturnValue(
        throwError(() => ({
          status: 500,
          error: {
            error: {
              code: 'INTERNAL_SERVER_ERROR',
              message: 'Koneksi database terputus',
            },
          },
        }))
      );

      component.resetForm.setValue({
        passwordBaru: 'password123',
        konfirmasiPassword: 'password123',
      });

      component.onSubmit();
      fixture.detectChanges();

      expect(component.hasTokenError()).toBe(false);
      expect(component.isSubmitted()).toBe(false);
      expect(component.errorMessage()).toBe('Koneksi database terputus');
      expect(toastSpy).toHaveBeenCalledWith('Koneksi database terputus');
    });
  });

  describe('Without token in queryParams', () => {
    beforeEach(async () => {
      await createComponent({});
    });

    it('should show token error screen immediately on load', () => {
      expect(component.hasTokenError()).toBe(true);

      const compiled = fixture.nativeElement as HTMLElement;
      expect(compiled.textContent).toContain('Link Tidak Valid atau Kadaluarsa');
      expect(compiled.textContent).toContain('Minta Link Baru');
    });
  });
});
