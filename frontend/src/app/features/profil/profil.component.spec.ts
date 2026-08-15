import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { toast } from '@spartan-ng/brain/sonner';
import { AuthService } from '../../core/auth/auth.service';
import { UserResponse } from '../../core/auth/auth.types';
import { ProfilComponent } from './profil.component';

describe('ProfilComponent', () => {
  let component: ProfilComponent;
  let fixture: ComponentFixture<ProfilComponent>;
  let authServiceSpy: {
    currentUser: ReturnType<typeof vi.fn>;
    changePassword: ReturnType<typeof vi.fn>;
  };

  const mockUser: UserResponse = {
    id: 1,
    nama: 'Dr. Budi Santoso',
    roles: ['dokter'],
  };

  beforeEach(async () => {
    authServiceSpy = {
      currentUser: vi.fn().mockReturnValue(mockUser),
      changePassword: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [ProfilComponent],
      providers: [{ provide: AuthService, useValue: authServiceSpy }],
    }).compileComponents();

    fixture = TestBed.createComponent(ProfilComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should render read-only user info from currentUser signal', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Dr. Budi Santoso');
    expect(compiled.textContent).toContain('dokter');
  });

  it('should validate passwordBaru min 8 characters and password confirmation mismatch', () => {
    const form = component.changePasswordForm;

    component.newPasswordControl.setValue('short');
    component.newPasswordControl.markAsTouched();
    expect(component.newPasswordControl.errors?.['minlength']).toBeTruthy();

    component.newPasswordControl.setValue('validPassword123');
    component.konfirmasiControl.setValue('differentPassword123');
    component.konfirmasiControl.markAsTouched();
    expect(form.errors?.['passwordsMismatch']).toBe(true);

    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Konfirmasi kata sandi tidak cocok');
  });

  it('should render inline error under passwordLama when authService.changePassword returns HTTP 400 INVALID_PASSWORD', () => {
    authServiceSpy.changePassword.mockReturnValue(
      throwError(() => ({
        status: 400,
        error: {
          error: {
            code: 'INVALID_PASSWORD',
            message: 'Kata sandi saat ini salah',
          },
        },
      }))
    );

    component.changePasswordForm.setValue({
      passwordLama: 'salahPassword',
      passwordBaru: 'passwordBaru123',
      konfirmasiPassword: 'passwordBaru123',
    });

    component.onSubmit();
    fixture.detectChanges();

    expect(authServiceSpy.changePassword).toHaveBeenCalledWith('salahPassword', 'passwordBaru123');
    expect(component.oldPasswordError()).toBe('Kata sandi saat ini salah');

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Kata sandi saat ini salah');
  });

  it('should trigger toast.error when technical HTTP 500 error occurs', () => {
    const toastSpy = vi.spyOn(toast, 'error').mockImplementation(() => '' as any);
    authServiceSpy.changePassword.mockReturnValue(
      throwError(() => ({
        status: 500,
        error: {
          error: {
            code: 'INTERNAL_SERVER_ERROR',
            message: 'Gagal memperbarui password di server',
          },
        },
      }))
    );

    component.changePasswordForm.setValue({
      passwordLama: 'passLama123',
      passwordBaru: 'passwordBaru123',
      konfirmasiPassword: 'passwordBaru123',
    });

    component.onSubmit();
    fixture.detectChanges();

    expect(component.oldPasswordError()).toBeNull();
    expect(toastSpy).toHaveBeenCalledWith('Gagal memperbarui password di server');
  });

  it('should call authService.changePassword, trigger toast.success, and reset form on valid submission', () => {
    const toastSpy = vi.spyOn(toast, 'success').mockImplementation(() => '' as any);
    authServiceSpy.changePassword.mockReturnValue(of(undefined));

    component.changePasswordForm.setValue({
      passwordLama: 'passLama123',
      passwordBaru: 'passwordBaru123',
      konfirmasiPassword: 'passwordBaru123',
    });

    component.onSubmit();
    fixture.detectChanges();

    expect(authServiceSpy.changePassword).toHaveBeenCalledWith('passLama123', 'passwordBaru123');
    expect(toastSpy).toHaveBeenCalledWith('Password berhasil diubah');

    // Form controls should be reset to empty/null
    expect(component.changePasswordForm.value.passwordLama).toBeFalsy();
    expect(component.changePasswordForm.value.passwordBaru).toBeFalsy();
  });
});
