import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { firstValueFrom, Observable, of } from 'rxjs';
import { AuthService } from '../auth/auth.service';
import { UserResponse } from '../auth/auth.types';
import { staffAuthResolver } from './staff-auth.resolver';

describe('staffAuthResolver', () => {
  let authServiceSpy: {
    isInitialized: ReturnType<typeof vi.fn>;
    currentUser: ReturnType<typeof vi.fn>;
    fetchMe: ReturnType<typeof vi.fn>;
  };

  const mockUser: UserResponse = {
    id: 1,
    nama: 'Petugas Test',
    roles: ['petugas'],
  };

  beforeEach(() => {
    authServiceSpy = {
      isInitialized: vi.fn(),
      currentUser: vi.fn(),
      fetchMe: vi.fn(),
    };

    TestBed.configureTestingModule({
      providers: [{ provide: AuthService, useValue: authServiceSpy }],
    });
  });

  it('should return currentUser immediately if authService is already initialized', async () => {
    authServiceSpy.isInitialized.mockReturnValue(true);
    authServiceSpy.currentUser.mockReturnValue(mockUser);

    const route = {} as ActivatedRouteSnapshot;
    const state = {} as RouterStateSnapshot;

    let result$!: Observable<UserResponse | null>;
    TestBed.runInInjectionContext(() => {
      result$ = staffAuthResolver(route, state) as Observable<UserResponse | null>;
    });

    const res = await firstValueFrom(result$);
    expect(res).toEqual(mockUser);
    expect(authServiceSpy.fetchMe).not.toHaveBeenCalled();
  });

  it('should call fetchMe() if authService is not yet initialized', async () => {
    authServiceSpy.isInitialized.mockReturnValue(false);
    authServiceSpy.fetchMe.mockReturnValue(of(mockUser));

    const route = {} as ActivatedRouteSnapshot;
    const state = {} as RouterStateSnapshot;

    let result$!: Observable<UserResponse | null>;
    TestBed.runInInjectionContext(() => {
      result$ = staffAuthResolver(route, state) as Observable<UserResponse | null>;
    });

    const res = await firstValueFrom(result$);
    expect(res).toEqual(mockUser);
    expect(authServiceSpy.fetchMe).toHaveBeenCalled();
  });
});
