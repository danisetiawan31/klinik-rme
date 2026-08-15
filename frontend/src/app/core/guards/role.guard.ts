import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { map, of } from 'rxjs';
import { AuthService } from '../auth/auth.service';

export const roleGuard = (...allowedRoles: string[]): CanActivateFn => {
  return () => {
    const authService = inject(AuthService);
    const router = inject(Router);

    const checkRole = (currentUser: any): boolean => {
      if (!currentUser) {
        router.navigate(['/login']);
        return false;
      }

      if (allowedRoles.length === 0) {
        return true;
      }

      const hasRole = allowedRoles.some((role) => currentUser.roles?.includes(role));

      if (hasRole) {
        return true;
      }

      // Redirect to /forbidden state if logged in but without required role (DESIGN.md §9.5)
      router.navigate(['/forbidden']);
      return false;
    };

    // If auth state is already initialized (e.g. navigation within SPA), check synchronously
    if (authService.isInitialized && authService.isInitialized()) {
      return checkRole(authService.currentUser());
    }

    // If page is hard-refreshed (F5), resolve auth state via fetchMe() first
    if (authService.fetchMe) {
      return authService.fetchMe().pipe(map((user) => checkRole(user)));
    }

    return checkRole(authService.currentUser());
  };
};
