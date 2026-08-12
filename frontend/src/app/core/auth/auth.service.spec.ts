import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { environment } from '../../../environments/environment';
import { AuthService } from './auth.service';
import { LoginRequest, LoginResponse, UserResponse } from './auth.types';

describe('AuthService', () => {
  let service: AuthService;
  let httpMock: HttpTestingController;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  beforeEach(() => {
    routerSpy = { navigate: vi.fn() };

    TestBed.configureTestingModule({
      providers: [
        AuthService,
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
      ],
    });

    service = TestBed.inject(AuthService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should initialize with default signals state', () => {
    expect(service.currentUser()).toBeNull();
    expect(service.isLoading()).toBe(false);
    expect(service.authError()).toBeNull();
    expect(service.isInitialized()).toBe(false);
  });

  describe('login()', () => {
    const mockCredentials: LoginRequest = {
      email: 'petugas@kliniksehat.id',
      password: 'password123',
    };

    const mockUser: UserResponse = {
      id: 1,
      nama: 'Petugas Test',
      roles: ['petugas'],
    };

    const mockLoginResponse: LoginResponse = {
      user: mockUser,
    };

    it('should set currentUser signal and clear authError on login success', () => {
      service.login(mockCredentials).subscribe((res) => {
        expect(res).toEqual(mockLoginResponse);
      });

      expect(service.isLoading()).toBe(true);

      const req = httpMock.expectOne(`${environment.apiUrl}/auth/login`);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual(mockCredentials);

      req.flush(mockLoginResponse);

      expect(service.currentUser()).toEqual(mockUser);
      expect(service.authError()).toBeNull();
      expect(service.isLoading()).toBe(false);
    });

    it('should set authError message and clear currentUser on login 401 failure', () => {
      const errorResponse = {
        error: {
          code: 'INVALID_CREDENTIALS',
          message: 'Email atau password salah',
          requestId: 'req-123',
        },
      };

      service.login(mockCredentials).subscribe({
        next: () => expect.fail('Should have failed with 401'),
        error: (err) => {
          const msg = err?.error?.message || err?.error?.error?.message;
          expect(msg).toBe('Email atau password salah');
        },
      });

      const req = httpMock.expectOne(`${environment.apiUrl}/auth/login`);
      req.flush(errorResponse, { status: 401, statusText: 'Unauthorized' });

      expect(service.currentUser()).toBeNull();
      expect(service.authError()).toBe('Email atau password salah');
      expect(service.isLoading()).toBe(false);
    });
  });

  describe('fetchMe()', () => {
    const mockUser: UserResponse = {
      id: 2,
      nama: 'Dokter Test',
      roles: ['dokter'],
    };

    it('should populate currentUser and set isInitialized to true on success', () => {
      service.fetchMe().subscribe((user) => {
        expect(user).toEqual(mockUser);
      });

      const req = httpMock.expectOne(`${environment.apiUrl}/auth/me`);
      expect(req.request.method).toBe('GET');
      req.flush(mockUser);

      expect(service.currentUser()).toEqual(mockUser);
      expect(service.isInitialized()).toBe(true);
    });

    it('should set currentUser to null and set isInitialized to true on failure', () => {
      service.fetchMe().subscribe((user) => {
        expect(user).toBeNull();
      });

      const req = httpMock.expectOne(`${environment.apiUrl}/auth/me`);
      req.flush({ error: { code: 'UNAUTHORIZED', message: 'Unauthorized', requestId: 'r2' } }, { status: 401, statusText: 'Unauthorized' });

      expect(service.currentUser()).toBeNull();
      expect(service.isInitialized()).toBe(true);
    });
  });

  describe('logout()', () => {
    it('should clear currentUser signal and navigate to /login', () => {
      service.currentUser.set({ id: 1, nama: 'User', roles: ['admin'] });

      service.logout().subscribe();

      const req = httpMock.expectOne(`${environment.apiUrl}/auth/logout`);
      expect(req.request.method).toBe('POST');
      req.flush(null, { status: 204, statusText: 'No Content' });

      expect(service.currentUser()).toBeNull();
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
    });
  });

  describe('getLandingRoute()', () => {
    it('should return /login for null user or empty roles', () => {
      expect(service.getLandingRoute(null)).toBe('/login');
      expect(service.getLandingRoute({ id: 1, nama: 'No Role', roles: [] })).toBe('/login');
    });

    it('should prioritize admin > dokter > petugas landing routes', () => {
      expect(service.getLandingRoute({ id: 1, nama: 'Admin', roles: ['admin'] })).toBe('/admin');
      expect(service.getLandingRoute({ id: 2, nama: 'Dokter', roles: ['dokter'] })).toBe('/antrian');
      expect(service.getLandingRoute({ id: 3, nama: 'Petugas', roles: ['petugas'] })).toBe('/antrian');

      // Multi-role priority: admin > dokter > petugas
      expect(service.getLandingRoute({ id: 4, nama: 'Admin Dokter', roles: ['admin', 'dokter'] })).toBe('/admin');
      expect(service.getLandingRoute({ id: 5, nama: 'Dokter Petugas', roles: ['petugas', 'dokter'] })).toBe('/antrian');
    });
  });
});
