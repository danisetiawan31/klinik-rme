import { HttpClient } from '@angular/common/http';
import { inject, Injectable, signal } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, finalize, Observable, of, tap, throwError } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ErrorEnvelope } from '../types/api-response.type';
import { LoginRequest, LoginResponse, UserResponse } from './auth.types';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private http = inject(HttpClient);
  private router = inject(Router);

  // State Signals
  readonly currentUser = signal<UserResponse | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly authError = signal<string | null>(null);
  readonly isInitialized = signal<boolean>(false);

  /**
   * Submit credentials to POST /api/v1/auth/login
   */
  login(credentials: LoginRequest): Observable<LoginResponse> {
    this.isLoading.set(true);
    this.authError.set(null);

    return this.http
      .post<LoginResponse>(`${environment.apiUrl}/auth/login`, credentials)
      .pipe(
        tap((res) => {
          this.currentUser.set(res.user);
          this.authError.set(null);
        }),
        catchError((err: any) => {
          let message = 'Email atau password salah';
          if (err?.error?.message) {
            message = err.error.message;
          } else if (err?.error?.error?.message) {
            message = err.error.error.message;
          }
          this.authError.set(message);
          this.currentUser.set(null);
          return throwError(() => err);
        }),
        finalize(() => {
          this.isLoading.set(false);
        })
      );
  }

  /**
   * Fetch logged in user info from GET /api/v1/auth/me
   */
  fetchMe(): Observable<UserResponse | null> {
    return this.http.get<UserResponse>(`${environment.apiUrl}/auth/me`).pipe(
      tap((user) => {
        this.currentUser.set(user);
        this.isInitialized.set(true);
      }),
      catchError(() => {
        this.currentUser.set(null);
        this.isInitialized.set(true);
        return of(null);
      })
    );
  }

  /**
   * Logout user via POST /api/v1/auth/logout
   */
  logout(): Observable<void> {
    return this.http.post<void>(`${environment.apiUrl}/auth/logout`, {}).pipe(
      finalize(() => {
        this.currentUser.set(null);
        this.router.navigate(['/login']);
      })
    );
  }

  /**
   * Send forgot-password request via POST /api/v1/auth/forgot-password [public]
   */
  forgotPassword(email: string): Observable<{ message: string }> {
    return this.http.post<{ message: string }>(
      `${environment.apiUrl}/auth/forgot-password`,
      { email }
    );
  }

  /**
   * Reset password via POST /api/v1/auth/reset-password [public]
   */
  resetPassword(token: string, passwordBaru: string): Observable<void> {
    return this.http.post<void>(`${environment.apiUrl}/auth/reset-password`, {
      token,
      passwordBaru,
    });
  }

  /**
   * Determine priority landing route based on roles: admin > dokter > petugas
   */
  getLandingRoute(user: UserResponse | null = this.currentUser()): string {
    if (!user || !user.roles || user.roles.length === 0) {
      return '/login';
    }
    if (user.roles.includes('admin')) {
      return '/admin';
    }
    if (user.roles.includes('dokter')) {
      return '/antrian';
    }
    if (user.roles.includes('petugas')) {
      return '/antrian';
    }
    return '/login';
  }
}
