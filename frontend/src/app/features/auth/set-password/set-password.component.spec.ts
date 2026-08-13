import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthService } from '../../../core/auth/auth.service';
import { SetPasswordComponent } from './set-password.component';

describe('SetPasswordComponent', () => {
  let component: SetPasswordComponent;
  let fixture: ComponentFixture<SetPasswordComponent>;
  let authServiceSpy: {
    resetPassword: ReturnType<typeof vi.fn>;
  };

  const setupTestBed = async (queryParams: Record<string, string> = { token: 'valid_token_123' }) => {
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
            snapshot: { queryParams },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(SetPasswordComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  };

  describe('With valid token in queryParams', () => {
    beforeEach(async () => {
      await setupTestBed({ token: 'valid_token_123' });
    });

    it('should validate password min 8 chars and confirmation mismatch', () => {
      const passwordCtrl = component.resetForm.controls.passwordBaru;
      const konfirmasiCtrl = component.resetForm.controls.konfirmasiPassword;

      // Password < 8 chars
      passwordCtrl.setValue('pass');
      passwordCtrl.markAsTouched();
      fixture.detectChanges();

      expect(passwordCtrl.valid).toBe(false);
      expect(passwordCtrl.errors?.['minlength']).toBeTruthy();
      expect(fixture.nativeElement.textContent).toContain('Password baru minimal 8 karakter');

      // Mismatch
      passwordCtrl.setValue('password123');
      konfirmasiCtrl.setValue('password999');
      konfirmasiCtrl.markAsTouched();
      component.resetForm.markAsTouched();
      fixture.detectChanges();

      expect(component.resetForm.errors?.['passwordsMismatch']).toBe(true);
      expect(fixture.nativeElement.textContent).toContain('Konfirmasi password tidak cocok');
    });

    it('should call authService.resetPassword and render Success State card on valid submit', () => {
      authServiceSpy.resetPassword.mockReturnValue(of(undefined));

      component.resetForm.setValue({
        passwordBaru: 'password123',
        konfirmasiPassword: 'password123',
      });

      component.onSubmit();
      fixture.detectChanges();

      expect(authServiceSpy.resetPassword).toHaveBeenCalledWith('valid_token_123', 'password123');
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
              message: 'Token reset/invite tidak valid, expired, atau sudah digunakan',
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

    it('should render ToastComponent error and keep form active when technical HTTP 500 error occurs', () => {
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

      const compiled = fixture.nativeElement as HTMLElement;
      expect(compiled.textContent).toContain('Koneksi database terputus');
    });
  });

  describe('Without token in queryParams', () => {
    beforeEach(async () => {
      await setupTestBed({});
    });

    it('should render Token Error State card immediately when token is missing', () => {
      expect(component.hasTokenError()).toBe(true);

      const compiled = fixture.nativeElement as HTMLElement;
      expect(compiled.textContent).toContain('Link Tidak Valid atau Kadaluarsa');
      expect(compiled.textContent).toContain('Minta Link Baru');
    });
  });
});
