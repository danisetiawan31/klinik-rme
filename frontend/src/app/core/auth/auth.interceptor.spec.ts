import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { authInterceptor } from './auth.interceptor';

describe('authInterceptor', () => {
  let http: HttpClient;
  let httpMock: HttpTestingController;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  beforeEach(() => {
    routerSpy = { navigate: vi.fn() };

    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([authInterceptor])),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
      ],
    });

    http = TestBed.inject(HttpClient);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should attach withCredentials: true to outgoing requests', () => {
    http.get('/api/v1/pasien').subscribe();

    const req = httpMock.expectOne('/api/v1/pasien');
    expect(req.request.withCredentials).toBe(true);
    req.flush([]);
  });

  it('should NOT redirect to /login on 401 from /auth/login', () => {
    http.post('/api/v1/auth/login', { email: 'a@b.c', password: 'p' }).subscribe({
      error: () => {},
    });

    const req = httpMock.expectOne('/api/v1/auth/login');
    req.flush(
      { error: { code: 'INVALID_CREDENTIALS', message: 'Email atau password salah', requestId: 'r1' } },
      { status: 401, statusText: 'Unauthorized' }
    );

    expect(routerSpy.navigate).not.toHaveBeenCalled();
  });

  it('should redirect to /login on 401 from regular endpoints (e.g. /api/v1/pasien)', () => {
    http.get('/api/v1/pasien').subscribe({
      error: () => {},
    });

    const req = httpMock.expectOne('/api/v1/pasien');
    req.flush(
      { error: { code: 'UNAUTHORIZED', message: 'Session expired', requestId: 'r2' } },
      { status: 401, statusText: 'Unauthorized' }
    );

    expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
  });

  it('should NOT redirect to /login on 401 if request has X-Display-Token header', () => {
    http.get('/api/v1/klinik/1/antrian', { headers: { 'X-Display-Token': 'tok123' } }).subscribe({
      error: () => {},
    });

    const req = httpMock.expectOne('/api/v1/klinik/1/antrian');
    req.flush(
      { error: { code: 'UNAUTHORIZED', message: 'Invalid display token', requestId: 'r3' } },
      { status: 401, statusText: 'Unauthorized' }
    );

    expect(routerSpy.navigate).not.toHaveBeenCalled();
  });
});
