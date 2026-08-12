import { inject } from '@angular/core';
import { ResolveFn } from '@angular/router';
import { of } from 'rxjs';
import { AuthService } from '../auth/auth.service';
import { UserResponse } from '../auth/auth.types';

export const staffAuthResolver: ResolveFn<UserResponse | null> = () => {
  const authService = inject(AuthService);

  if (authService.isInitialized()) {
    return of(authService.currentUser());
  }

  return authService.fetchMe();
};
