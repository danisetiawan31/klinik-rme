import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot } from '@angular/router';
import { of } from 'rxjs';
import { AuthService } from '../auth/auth.service';
import { UserResponse } from '../auth/auth.types';
import { roleGuard } from './role.guard';

describe('roleGuard', () => {
  let authServiceSpy: {
    currentUser: ReturnType<typeof vi.fn>;
    isInitialized: ReturnType<typeof vi.fn>;
    fetchMe: ReturnType<typeof vi.fn>;
    getLandingRoute: ReturnType<typeof vi.fn>;
  };
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  beforeEach(() => {
    authServiceSpy = {
      currentUser: vi.fn(),
      isInitialized: vi.fn().mockReturnValue(true),
      fetchMe: vi.fn().mockReturnValue(of(null)),
      getLandingRoute: vi.fn(),
    };
    routerSpy = { navigate: vi.fn() };

    TestBed.configureTestingModule({
      providers: [
        { provide: AuthService, useValue: authServiceSpy },
        { provide: Router, useValue: routerSpy },
      ],
    });
  });

  it('should redirect to /login and return false if currentUser is null when initialized', () => {
    authServiceSpy.currentUser.mockReturnValue(null);

    const guard = roleGuard('petugas', 'dokter');
    const route = {} as ActivatedRouteSnapshot;
    const state = {} as RouterStateSnapshot;

    TestBed.runInInjectionContext(() => {
      const result = guard(route, state);
      expect(result).toBe(false);
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
    });
  });

  it('should return true if currentUser has at least one allowed role when initialized', () => {
    const mockUser: UserResponse = { id: 1, nama: 'Dokter', roles: ['dokter'] };
    authServiceSpy.currentUser.mockReturnValue(mockUser);

    const guard = roleGuard('petugas', 'dokter');
    const route = {} as ActivatedRouteSnapshot;
    const state = {} as RouterStateSnapshot;

    TestBed.runInInjectionContext(() => {
      const result = guard(route, state);
      expect(result).toBe(true);
      expect(routerSpy.navigate).not.toHaveBeenCalled();
    });
  });

  it('should resolve user asynchronously via fetchMe on hard refresh when uninitialized', () => {
    authServiceSpy.isInitialized.mockReturnValue(false);
    const mockUser: UserResponse = { id: 1, nama: 'Dokter', roles: ['dokter'] };
    authServiceSpy.fetchMe.mockReturnValue(of(mockUser));

    const guard = roleGuard('petugas', 'dokter');
    const route = {} as ActivatedRouteSnapshot;
    const state = {} as RouterStateSnapshot;

    TestBed.runInInjectionContext(() => {
      const result$ = guard(route, state) as any;
      result$.subscribe((allowed: boolean) => {
        expect(allowed).toBe(true);
        expect(authServiceSpy.fetchMe).toHaveBeenCalled();
        expect(routerSpy.navigate).not.toHaveBeenCalled();
      });
    });
  });

  it('should redirect to /forbidden and return false if currentUser lacks allowed roles', () => {
    const mockUser: UserResponse = { id: 1, nama: 'Petugas', roles: ['petugas'] };
    authServiceSpy.currentUser.mockReturnValue(mockUser);

    const guard = roleGuard('admin');
    const route = {} as ActivatedRouteSnapshot;
    const state = {} as RouterStateSnapshot;

    TestBed.runInInjectionContext(() => {
      const result = guard(route, state);
      expect(result).toBe(false);
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/forbidden']);
    });
  });
});
