import { provideRouter, Router } from '@angular/router';
import { TestBed } from '@angular/core/testing';
import { rekamMedisRoutes } from './rekam-medis.routes';

describe('rekamMedisRoutes', () => {
  let router: Router;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      providers: [provideRouter(rekamMedisRoutes)],
    }).compileComponents();

    router = TestBed.inject(Router);
  });

  it('configures routes for examination form and leaf detail view', () => {
    const config = router.config;

    const rootRoute = config.find((r) => r.path === '');
    const formRoute = config.find((r) => r.path === 'pemeriksaan/:kunjunganId');
    const detailRoute = config.find((r) => r.path === 'kunjungan/:kunjunganId');

    expect(rootRoute).toBeTruthy();
    expect(rootRoute?.redirectTo).toBe('/antrian');

    expect(formRoute).toBeTruthy();
    expect(formRoute?.loadComponent).toBeTruthy();

    expect(detailRoute).toBeTruthy();
    expect(detailRoute?.loadComponent).toBeTruthy();
  });
});
