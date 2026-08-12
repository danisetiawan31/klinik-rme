import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../auth/auth.service';

export const roleGuard = (...allowedRoles: string[]): CanActivateFn => {
  return () => {
    const authService = inject(AuthService);
    const router = inject(Router);
    const currentUser = authService.currentUser();

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

    // Redirect to landing route if logged in but without required role
    const landingRoute = authService.getLandingRoute(currentUser);
    router.navigate([landingRoute]);
    return false;
  };
};
