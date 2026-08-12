import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { of } from 'rxjs';
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
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    authErrorSignal = signal<string | null>(null);
    isLoadingSignal = signal<boolean>(false);

    authServiceSpy = {
      login: vi.fn(),
      getLandingRoute: vi.fn(),
      isLoading: isLoadingSignal,
      authError: authErrorSignal,
    };
    routerSpy = { navigate: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [LoginComponent],
      providers: [
        { provide: AuthService, useValue: authServiceSpy },
        { provide: Router, useValue: routerSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LoginComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
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

  it('should render error toast when authError signal has a value', () => {
    authErrorSignal.set('Email atau password salah');
    fixture.detectChanges();

    const toastEl = fixture.nativeElement.querySelector('app-toast');
    expect(toastEl).toBeTruthy();
    expect(fixture.nativeElement.textContent).toContain('Email atau password salah');
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
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/antrian']);
  });
});
