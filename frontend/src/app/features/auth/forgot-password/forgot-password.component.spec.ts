import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthService } from '../../../core/auth/auth.service';
import { ForgotPasswordComponent } from './forgot-password.component';

describe('ForgotPasswordComponent', () => {
  let component: ForgotPasswordComponent;
  let fixture: ComponentFixture<ForgotPasswordComponent>;
  let authServiceSpy: {
    forgotPassword: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    authServiceSpy = {
      forgotPassword: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [ForgotPasswordComponent],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authServiceSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ForgotPasswordComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should validate email field as required when empty', () => {
    const emailControl = component.forgotForm.controls.email;

    emailControl.setValue('');
    emailControl.markAsTouched();
    fixture.detectChanges();

    expect(emailControl.valid).toBe(false);
    expect(emailControl.errors?.['required']).toBe(true);

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Email wajib diisi');
  });

  it('should validate email format when invalid email string provided', () => {
    const emailControl = component.forgotForm.controls.email;

    emailControl.setValue('invalid-email-string');
    emailControl.markAsTouched();
    fixture.detectChanges();

    expect(emailControl.valid).toBe(false);
    expect(emailControl.errors?.['email']).toBe(true);

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Format email tidak valid');
  });

  it('should trigger authService.forgotPassword and swap to generic success state on submit', () => {
    authServiceSpy.forgotPassword.mockReturnValue(
      of({ message: 'Jika email terdaftar, instruksi reset password telah dikirim' })
    );

    component.forgotForm.controls.email.setValue('dokter@klinik.com');
    component.onSubmit();
    fixture.detectChanges();

    expect(authServiceSpy.forgotPassword).toHaveBeenCalledWith('dokter@klinik.com');
    expect(component.isSubmitted()).toBe(true);
    expect(component.submittedEmail()).toBe('dokter@klinik.com');

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Instruksi Dikirim');
    expect(compiled.textContent).toContain('dokter@klinik.com');
    expect(compiled.textContent).toContain('Kembali ke Halaman Login');
  });

  it('should display error Toast and remain on form state when technical HTTP error occurs', () => {
    authServiceSpy.forgotPassword.mockReturnValue(
      throwError(() => ({ error: { message: 'Server backend mengalami gangguan' } }))
    );

    component.forgotForm.controls.email.setValue('staf@klinik.com');
    component.onSubmit();
    fixture.detectChanges();

    expect(authServiceSpy.forgotPassword).toHaveBeenCalledWith('staf@klinik.com');
    expect(component.isSubmitted()).toBe(false);
    expect(component.errorMessage()).toBe('Server backend mengalami gangguan');

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Server backend mengalami gangguan');
  });
});
