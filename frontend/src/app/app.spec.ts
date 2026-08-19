import { TestBed } from '@angular/core/testing';
import { App } from './app';

describe('App', () => {
  beforeEach(async () => {
    if (!window.matchMedia) {
      window.matchMedia = () =>
        ({
          matches: false,
          addListener: () => {},
          removeListener: () => {},
          addEventListener: () => {},
          removeEventListener: () => {},
          dispatchEvent: () => false,
        }) as unknown as MediaQueryList;
    }

    await TestBed.configureTestingModule({
      imports: [App],
    }).compileComponents();
  });

  it('should create the app', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    expect(app).toBeTruthy();
  });

  it('should render router outlet', () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('router-outlet')).toBeTruthy();
  });

  it('should have public papan-antrian route configured without staff guards', async () => {
    const { routes } = await import('./app.routes');
    const papanRoute = routes.find((r) => r.path === 'papan-antrian');

    expect(papanRoute).toBeTruthy();
    expect(papanRoute?.canActivate).toBeUndefined();
    expect(papanRoute?.resolve).toBeUndefined();
    expect(papanRoute?.loadComponent).toBeTruthy();
  });

  it('should have admin route configured with role guard', async () => {
    const { routes } = await import('./app.routes');
    const shellRoute = routes.find((r) => r.path === '');
    const adminRoute = shellRoute?.children?.find((r) => r.path === 'admin');

    expect(adminRoute).toBeTruthy();
    expect(adminRoute?.canActivate).toBeDefined();
    expect(adminRoute?.loadComponent).toBeTruthy();
  });
});
