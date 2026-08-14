import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthService } from '../../core/auth/auth.service';
import { UserResponse } from '../../core/auth/auth.types';
import { ProfilComponent } from './profil.component';

describe('ProfilComponent', () => {
  let component: ProfilComponent;
  let fixture: ComponentFixture<ProfilComponent>;
  let currentUserSignal: WritableSignal<UserResponse | null>;
  let authServiceSpy: {
    changePassword: ReturnType<typeof vi.fn>;
    currentUser: WritableSignal<UserResponse | null>;
  };

  beforeEach(async () => {
    currentUserSignal = signal<UserResponse | null>({
      id: 10,
      nama: 'Dr. Budi Santoso',
      roles: ['dokter'],
    });

    authServiceSpy = {
      changePassword: vi.fn(),
      currentUser: currentUserSignal,
    };

    await TestBed.configureTestingModule({
      imports: [ProfilComponent],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authServiceSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ProfilComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should render read-only user info from currentUser signal', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Dr. Budi Santoso');
    expect(compiled.textContent).toContain('dokter');
  });

  it('should validate passwordBaru min 8 characters and password confirmation mismatch', () => {
    const newPasswordCtrl = component.changePasswordForm.controls.passwordBaru;
    const konfirmasiCtrl = component.changePasswordForm.controls.konfirmasiPassword;

    // Minlength test
    newPasswordCtrl.setValue('short');
    newPasswordCtrl.markAsTouched();
    fixture.detectChanges();

    expect(newPasswordCtrl.valid).toBe(false);
    expect(newPasswordCtrl.errors?.['minlength']).toBeTruthy();
    expect(fixture.nativeElement.textContent).toContain('Kata sandi baru minimal 8 karakter');

    // Mismatch test
    newPasswordCtrl.setValue('password123');
    konfirmasiCtrl.setValue('password888');
    konfirmasiCtrl.markAsTouched();
    component.changePasswordForm.markAsTouched();
    fixture.detectChanges();

    expect(component.changePasswordForm.errors?.['passwordsMismatch']).toBe(true);
    expect(fixture.nativeElement.textContent).toContain('Konfirmasi kata sandi tidak cocok');
  });

  it('should render inline error under passwordLama when authService.changePassword returns HTTP 400 INVALID_PASSWORD', () => {
    authServiceSpy.changePassword.mockReturnValue(
      throwError(() => ({
        status: 400,
        error: {
          error: {
            code: 'INVALID_PASSWORD',
            message: 'Password lama tidak sesuai',
          },
        },
      }))
    );

    component.changePasswordForm.setValue({
      passwordLama: 'passSalah123',
      passwordBaru: 'passwordBaru123',
      konfirmasiPassword: 'passwordBaru123',
    });

    component.onSubmit();
    fixture.detectChanges();

    expect(authServiceSpy.changePassword).toHaveBeenCalledWith('passSalah123', 'passwordBaru123');
    expect(component.oldPasswordError()).toBe('Password lama tidak sesuai');

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Password lama tidak sesuai');
  });

  it('should render ToastComponent error when technical HTTP 500 error occurs', () => {
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
    expect(component.toastType()).toBe('error');
    expect(component.toastMessage()).toBe('Gagal memperbarui password di server');

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Gagal memperbarui password di server');
  });

  it('should call authService.changePassword, render success ToastComponent, and reset form on valid submission', () => {
    authServiceSpy.changePassword.mockReturnValue(of(undefined));

    component.changePasswordForm.setValue({
      passwordLama: 'passLama123',
      passwordBaru: 'passwordBaru123',
      konfirmasiPassword: 'passwordBaru123',
    });

    component.onSubmit();
    fixture.detectChanges();

    expect(authServiceSpy.changePassword).toHaveBeenCalledWith('passLama123', 'passwordBaru123');
    expect(component.toastType()).toBe('success');
    expect(component.toastMessage()).toBe('Password berhasil diubah');

    // Form controls should be reset to empty/null
    expect(component.changePasswordForm.value.passwordLama).toBeFalsy();
    expect(component.changePasswordForm.value.passwordBaru).toBeFalsy();
  });
});
