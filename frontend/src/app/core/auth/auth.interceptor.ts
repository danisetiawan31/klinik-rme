import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { ErrorEnvelope } from '../types/api-response.type';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const router = inject(Router);

  // Attach withCredentials: true
  const modifiedReq = req.clone({
    withCredentials: true,
  });

  return next(modifiedReq).pipe(
    catchError((error: HttpErrorResponse) => {
      // Handle 401 Unauthorized
      if (error.status === 401) {
        const isDisplayTokenReq = req.headers.has('X-Display-Token');
        const isAuthEndpoint =
          req.url.includes('/auth/login') ||
          req.url.includes('/auth/forgot-password') ||
          req.url.includes('/auth/reset-password') ||
          req.url.includes('/auth/me');

        if (!isDisplayTokenReq && !isAuthEndpoint) {
          router.navigate(['/login']);
        }
      }

      // Parse error body to ErrorEnvelope if possible
      let parsedError: ErrorEnvelope | null = null;
      if (error.error && typeof error.error === 'object' && 'error' in error.error) {
        parsedError = error.error as ErrorEnvelope;
      }

      // We throw the parsed error if it matches the contract, otherwise the original error
      return throwError(() => parsedError || error);
    })
  );
};
