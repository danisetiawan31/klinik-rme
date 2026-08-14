import { provideRouter, Router } from '@angular/router';
import { TestBed } from '@angular/core/testing';
import { pasienRoutes } from './pasien.routes';
import { AuthService } from '../../core/auth/auth.service';
import { signal } from '@angular/core';

const mockAuthService = {
  currentUser: signal({ id: 1, nama: 'Petugas', roles: ['petugas'] }),
};

describe('pasienRoutes (Routing Collision Test)', () => {
  let router: Router;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      providers: [
        provideRouter(pasienRoutes),
        { provide: AuthService, useValue: mockAuthService },
      ],
    }).compileComponents();

    router = TestBed.inject(Router);
  });

  it('resolves /baru to PasienFormComponent, not PasienDetailComponent', async () => {
    const config = router.config;

    // Find route 'baru' and route ':id'
    const baruRouteIndex = config.findIndex((r) => r.path === 'baru');
    const idRouteIndex = config.findIndex((r) => r.path === ':id');

    expect(baruRouteIndex).toBeGreaterThan(-1);
    expect(idRouteIndex).toBeGreaterThan(-1);

    // Static route 'baru' MUST appear before parameterized route ':id'
    expect(baruRouteIndex).toBeLessThan(idRouteIndex);
  });

  it('enforces RBAC guards on all pasien routes', () => {
    const searchRoute = router.config.find((r) => r.path === '');
    const baruRoute = router.config.find((r) => r.path === 'baru');
    const detailRoute = router.config.find((r) => r.path === ':id');

    expect(searchRoute?.canActivate).toBeTruthy();
    expect(baruRoute?.canActivate).toBeTruthy();
    expect(detailRoute?.canActivate).toBeTruthy();
  });
});
