import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { toast } from '@spartan-ng/brain/sonner';
import { AuthService } from '../../../core/auth/auth.service';
import { LoginComponent } from './login.component';

describe('LoginComponent', () => {
  let fixture: ComponentFixture<LoginComponent>;
  let component: LoginComponent;
  let authErrorSignal: WritableSignal<string | null>;
  let isLoadingSignal: WritableSignal<boolean>;
  let authServiceSpy: {
    login: ReturnType<typeof vi.fn>;
    getLandingRoute: ReturnType<typeof vi.fn>;
    isLoading: WritableSignal<boolean>;
    authError: WritableSignal<string | null>;
  };
  let router: Router;

  beforeEach(async () => {
    authErrorSignal = signal<string | null>(null);
    isLoadingSignal = signal<boolean>(false);

    authServiceSpy = {
      login: vi.fn(),
      getLandingRoute: vi.fn(),
      isLoading: isLoadingSignal,
      authError: authErrorSignal,
    };

    await TestBed.configureTestingModule({
      imports: [LoginComponent],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authServiceSpy },
      ],
    }).compileComponents();

    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigate').mockImplementation(() => Promise.resolve(true));

    fixture = TestBed.createComponent(LoginComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should render default login form state', () => {
    const heading = fixture.nativeElement.querySelector('h1');
    expect(heading.textContent).toContain('Masuk');

    const emailInput = fixture.nativeElement.querySelector('input[type="email"]');
    const passwordComponent = fixture.nativeElement.querySelector('app-sensitive-value');
    const submitBtn = fixture.nativeElement.querySelector('button[type="submit"]');

    expect(emailInput).toBeTruthy();
    expect(passwordComponent).toBeTruthy();
    expect(submitBtn).toBeTruthy();
  });

  it('should trigger toast.error when login fails', () => {
    const toastSpy = vi.spyOn(toast, 'error').mockImplementation(() => '' as any);
    authServiceSpy.login.mockReturnValue(throwError(() => new Error('error')));
    authErrorSignal.set('Email atau password salah');

    component.loginForm.setValue({
      email: 'petugas@kliniksehat.id',
      password: 'password123',
    });

    component.onSubmit();

    expect(toastSpy).toHaveBeenCalledWith('Email atau password salah');
  });

  it('should render loading spinner and disable submit button when isLoading is true', () => {
    isLoadingSignal.set(true);
    fixture.detectChanges();

    const submitBtn = fixture.nativeElement.querySelector('button[type="submit"]') as HTMLButtonElement;
    expect(submitBtn.disabled).toBe(true);

    const spinner = fixture.nativeElement.querySelector('svg.kl-spinner');
    expect(spinner).toBeTruthy();
  });

  it('should call authService.login and navigate to landing route on valid submission', () => {
    const mockUser = { id: 1, nama: 'Petugas', roles: ['petugas'] };
    authServiceSpy.login.mockReturnValue(of({ user: mockUser }));
    authServiceSpy.getLandingRoute.mockReturnValue('/antrian');

    component.loginForm.setValue({
      email: 'petugas@kliniksehat.id',
      password: 'password123',
    });

    component.onSubmit();

    expect(authServiceSpy.login).toHaveBeenCalledWith({
      email: 'petugas@kliniksehat.id',
      password: 'password123',
    });
    expect(router.navigate).toHaveBeenCalledWith(['/antrian']);
  });
});
