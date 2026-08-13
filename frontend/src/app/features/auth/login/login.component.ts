import {
  ChangeDetectionStrategy,
  Component,
  inject,
} from '@angular/core';
import {
  FormBuilder,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { SensitiveValueComponent } from '../../../shared/components/sensitive-value/sensitive-value.component';
import { ToastComponent } from '../../../shared/components/toast/toast.component';

@Component({
  selector: 'app-login',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, RouterLink, SensitiveValueComponent, ToastComponent],
  template: `
    <!-- ── Toast error top-center ── -->
    <app-toast
      [message]="authService.authError() || ''"
      type="error"
      (dismiss)="onDismissToast()"
    />

    <!-- ── Page background ── -->
    <div
      class="min-h-[100dvh] w-full flex flex-col items-center justify-center px-4 py-10"
      style="
        background-color: #F0FDFA;
        background-image: radial-gradient(ellipse 90% 55% at 50% 0%, rgba(8,145,178,0.10) 0%, transparent 65%);
      "
    >
      <!-- ── Brand ── -->
      <div class="flex flex-col items-center text-center mb-8">
        <!-- Icon badge -->
        <div
          class="flex items-center justify-center mb-4"
          style="
            width:54px; height:54px;
            border: 2px solid #0891B2;
            border-radius: 12px;
            background: #fff;
            box-shadow: 0 1px 4px rgba(8,145,178,0.15);
          "
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="26" height="26"
            viewBox="0 0 24 24" fill="none"
            stroke="#0891B2" stroke-width="2.5"
            stroke-linecap="round" stroke-linejoin="round"
            aria-hidden="true">
            <path d="M12 5v14M5 12h14"/>
          </svg>
        </div>

        <!-- Brand name -->
        <p style="font-family:var(--font-heading); font-size:22px; font-weight:700; color:#0891B2; letter-spacing:-0.02em; line-height:1.2;">
          Klinik Sehat
        </p>
        <p style="font-family:var(--font-body); font-size:13px; font-weight:500; color:#64748B; margin-top:3px;">
          RME &amp; Antrian
        </p>
      </div>

      <!-- ── Card ── -->
      <div
        class="w-full"
        style="
          max-width:388px;
          background:#FFFFFF;
          border:1px solid #CCFBF1;
          border-radius:12px;
          box-shadow:0 4px 6px rgba(0,0,0,0.07);
          padding:32px 28px 28px;
        "
      >
        <!-- Card header -->
        <h1 style="font-family:var(--font-heading); font-size:20px; font-weight:700; color:#0F172A; margin-bottom:4px;">
          Masuk
        </h1>
        <p style="font-family:var(--font-body); font-size:13px; color:#64748B; line-height:1.5; margin-bottom:24px;">
          Silakan masuk untuk mengakses sistem internal klinik.
        </p>

        <!-- Form -->
        <form [formGroup]="loginForm" (ngSubmit)="onSubmit()" novalidate style="display:flex;flex-direction:column;gap:16px;">

          <!-- Email -->
          <div style="display:flex;flex-direction:column;gap:5px;">
            <label for="login-email" style="font-family:var(--font-body);font-size:12.5px;font-weight:600;color:#334155;">
              Email
            </label>
            <input
              id="login-email"
              type="email"
              formControlName="email"
              placeholder="Masukkan email"
              class="kl-input"
              autocomplete="username email"
              [attr.aria-invalid]="loginForm.controls.email.touched && loginForm.controls.email.invalid ? 'true' : null"
            />
            @if (loginForm.controls.email.touched && loginForm.controls.email.errors?.['required']) {
              <span style="font-size:11.5px;color:#DC2626;" role="alert">Email wajib diisi</span>
            }
            @if (loginForm.controls.email.touched && loginForm.controls.email.errors?.['email']) {
              <span style="font-size:11.5px;color:#DC2626;" role="alert">Format email tidak valid</span>
            }
          </div>

          <!-- Password -->
          <div style="display:flex;flex-direction:column;gap:5px;">
            <label for="sv-password" style="font-family:var(--font-body);font-size:12.5px;font-weight:600;color:#334155;">
              Password
            </label>
            <app-sensitive-value
              id="sv-password"
              mode="input"
              formControlName="password"
              placeholder="Masukkan password"
            />
            @if (loginForm.controls.password.touched && loginForm.controls.password.errors?.['required']) {
              <span style="font-size:11.5px;color:#DC2626;" role="alert">Password wajib diisi</span>
            }
          </div>

          <!-- Submit -->
          <button
            type="submit"
            class="kl-btn-primary"
            style="margin-top:4px;"
            [disabled]="isLoading()"
            [attr.aria-busy]="isLoading() ? 'true' : null"
          >
            @if (isLoading()) {
              <svg class="kl-spinner" xmlns="http://www.w3.org/2000/svg"
                width="16" height="16" viewBox="0 0 24 24"
                fill="none" stroke="currentColor" stroke-width="2.5"
                stroke-linecap="round" aria-hidden="true">
                <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
              </svg>
            }
            Masuk
          </button>
        </form>

        <!-- Forgot -->
        <div style="text-align:center; margin-top:20px;">
          <a
            routerLink="/forgot-password"
            style="font-family:var(--font-body);font-size:13px;font-weight:600;color:#0891B2;cursor:pointer;text-decoration:none;transition:color 150ms;"
            onmouseenter="this.style.textDecoration='underline'"
            onmouseleave="this.style.textDecoration='none'"
          >
            Lupa password?
          </a>
        </div>
      </div>
    </div>
  `,
})
export class LoginComponent {
  readonly authService = inject(AuthService);
  private fb = inject(FormBuilder);
  private router = inject(Router);

  readonly isLoading = this.authService.isLoading;

  readonly loginForm = this.fb.group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', [Validators.required]],
  });

  onSubmit(): void {
    if (this.loginForm.invalid) {
      this.loginForm.markAllAsTouched();
      return;
    }
    const { email, password } = this.loginForm.value;
    if (!email || !password) return;

    this.loginForm.disable();
    this.authService.login({ email, password }).subscribe({
      next: (res) => {
        this.loginForm.enable();
        this.router.navigate([this.authService.getLandingRoute(res.user)]);
      },
      error: () => {
        this.loginForm.enable();
      },
    });
  }

  onDismissToast(): void {
    this.authService.authError.set(null);
  }
}
