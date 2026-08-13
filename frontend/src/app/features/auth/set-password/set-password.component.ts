import {
  ChangeDetectionStrategy,
  Component,
  OnInit,
  inject,
  signal,
} from '@angular/core';
import {
  AbstractControl,
  FormBuilder,
  ReactiveFormsModule,
  ValidationErrors,
  ValidatorFn,
  Validators,
} from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { ErrorEnvelope } from '../../../core/types/api-response.type';
import { SensitiveValueComponent } from '../../../shared/components/sensitive-value/sensitive-value.component';
import { ToastComponent } from '../../../shared/components/toast/toast.component';

const passwordsMatchValidator: ValidatorFn = (
  control: AbstractControl
): ValidationErrors | null => {
  const p1 = control.get('passwordBaru')?.value;
  const p2 = control.get('konfirmasiPassword')?.value;
  if (p1 && p2 && p1 !== p2) {
    return { passwordsMismatch: true };
  }
  return null;
};

@Component({
  selector: 'app-set-password',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, RouterLink, SensitiveValueComponent, ToastComponent],
  template: `
    <!-- ── Toast error top-center untuk kegagalan teknis (500/Network/Timeout) ── -->
    @if (errorMessage()) {
      <app-toast
        [message]="errorMessage() || ''"
        type="error"
        (dismiss)="onDismissToast()"
      />
    }

    <!-- ── Page background (Hero Zone - DESIGN.md §1.1) ── -->
    <div
      class="min-h-[100dvh] w-full flex flex-col items-center justify-center px-4 py-10"
      style="
        background-color: var(--color-background);
        background-image: radial-gradient(ellipse 90% 55% at 50% 0%, rgba(8,145,178,0.10) 0%, transparent 65%);
      "
    >
      <!-- ── Brand Header ── -->
      <div class="flex flex-col items-center text-center mb-8">
        <div
          class="flex items-center justify-center mb-4"
          style="
            width:54px; height:54px;
            border: 2px solid var(--color-primary);
            border-radius: var(--radius-lg);
            background: var(--color-card);
            box-shadow: var(--shadow-1);
          "
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="26" height="26"
            viewBox="0 0 24 24" fill="none"
            stroke="var(--color-primary)" stroke-width="2.5"
            stroke-linecap="round" stroke-linejoin="round"
            aria-hidden="true">
            <path d="M12 5v14M5 12h14"/>
          </svg>
        </div>

        <p style="font-family:var(--font-heading); font-size:22px; font-weight:700; color:var(--color-primary); letter-spacing:-0.02em; line-height:1.2;">
          Klinik Sehat
        </p>
        <p style="font-family:var(--font-body); font-size:13px; font-weight:500; color:var(--color-muted-foreground); margin-top:3px;">
          RME &amp; Antrian
        </p>
      </div>

      <!-- ── Card ── -->
      <div
        class="w-full"
        style="
          max-width:388px;
          background:var(--color-card);
          border:1px solid var(--color-border);
          border-radius:var(--radius-lg);
          box-shadow:var(--shadow-2);
          padding:32px 28px 28px;
        "
      >
        @if (hasTokenError()) {
          <!-- Card State 1: Token Invalid / Expired / Missing -->
          <div class="flex flex-col items-center text-center space-y-4">
            <div
              class="h-12 w-12 rounded-full flex items-center justify-center"
              style="background: #FEF2F2; border: 1px solid #FCA5A5; color: var(--color-destructive);"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="12"/>
                <line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
            </div>

            <div>
              <h1 style="font-family:var(--font-heading); font-size:18px; font-weight:700; color:var(--color-foreground);">
                Link Tidak Valid atau Kadaluarsa
              </h1>
              <p style="font-family:var(--font-body); font-size:13px; color:var(--color-muted-foreground); line-height:1.6; margin-top:8px;">
                Link reset password atau undangan akun ini sudah tidak berlaku, expired, atau telah digunakan.
              </p>
            </div>

            <div class="p-3 rounded-[var(--radius-md)] text-left w-full" style="background:var(--color-muted); border:1px solid var(--color-border);">
              <p style="font-family:var(--font-body); font-size:12px; color:var(--color-muted-foreground); line-height:1.5;">
                Silakan minta link reset password baru melalui halaman Lupa Password.
              </p>
            </div>

            <a
              routerLink="/forgot-password"
              class="kl-btn-primary w-full inline-flex justify-center items-center gap-2"
              style="text-decoration:none; margin-top:8px;"
            >
              Minta Link Baru
            </a>
          </div>
        } @else if (isSubmitted()) {
          <!-- Card State 2: Sukses Reset Password -->
          <div class="flex flex-col items-center text-center space-y-4">
            <div
              class="h-12 w-12 rounded-full flex items-center justify-center"
              style="background: #F0FDF4; border: 1px solid #BBF7D0; color: var(--color-accent);"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m5 12 5 5L20 7"/>
              </svg>
            </div>

            <div>
              <h1 style="font-family:var(--font-heading); font-size:18px; font-weight:700; color:var(--color-foreground);">
                Password Berhasil Diubah
              </h1>
              <p style="font-family:var(--font-body); font-size:13px; color:var(--color-muted-foreground); line-height:1.6; margin-top:8px;">
                Password Anda telah berhasil diperbarui. Silakan masuk menggunakan password baru Anda.
              </p>
            </div>

            <a
              routerLink="/login"
              class="kl-btn-primary w-full inline-flex justify-center items-center gap-2"
              style="text-decoration:none; margin-top:8px;"
            >
              Masuk ke Halaman Login
            </a>
          </div>
        } @else {
          <!-- Card State 3: Active Form -->
          <h1 style="font-family:var(--font-heading); font-size:20px; font-weight:700; color:var(--color-foreground); margin-bottom:4px;">
            Atur Password Baru
          </h1>
          <p style="font-family:var(--font-body); font-size:13px; color:var(--color-muted-foreground); line-height:1.5; margin-bottom:24px;">
            Silakan buat password baru minimal 8 karakter untuk akun Anda.
          </p>

          <form [formGroup]="resetForm" (ngSubmit)="onSubmit()" novalidate style="display:flex;flex-direction:column;gap:16px;">
            <!-- Password Baru -->
            <div style="display:flex;flex-direction:column;gap:5px;">
              <label for="sv-password-baru" style="font-family:var(--font-body);font-size:12.5px;font-weight:600;color:var(--color-foreground);">
                Password Baru
              </label>
              <app-sensitive-value
                id="sv-password-baru"
                mode="input"
                formControlName="passwordBaru"
                placeholder="Masukkan password baru (min. 8 karakter)"
              />
              @if (passwordControl.touched && passwordControl.errors?.['required']) {
                <span style="font-size:11.5px;color:var(--color-destructive);" role="alert">Password baru wajib diisi</span>
              }
              @if (passwordControl.touched && passwordControl.errors?.['minlength']) {
                <span style="font-size:11.5px;color:var(--color-destructive);" role="alert">Password baru minimal 8 karakter</span>
              }
            </div>

            <!-- Konfirmasi Password -->
            <div style="display:flex;flex-direction:column;gap:5px;">
              <label for="sv-konfirmasi-password" style="font-family:var(--font-body);font-size:12.5px;font-weight:600;color:var(--color-foreground);">
                Konfirmasi Password Baru
              </label>
              <app-sensitive-value
                id="sv-konfirmasi-password"
                mode="input"
                formControlName="konfirmasiPassword"
                placeholder="Ulangi password baru"
              />
              @if (konfirmasiControl.touched && konfirmasiControl.errors?.['required']) {
                <span style="font-size:11.5px;color:var(--color-destructive);" role="alert">Konfirmasi password wajib diisi</span>
              }
              @if ((resetForm.touched || konfirmasiControl.touched) && resetForm.errors?.['passwordsMismatch'] && !konfirmasiControl.errors?.['required']) {
                <span style="font-size:11.5px;color:var(--color-destructive);" role="alert">Konfirmasi password tidak cocok</span>
              }
            </div>

            <!-- Submit Button -->
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
              Simpan Password Baru
            </button>
          </form>

          <!-- Back to login link -->
          <div style="text-align:center; margin-top:20px;">
            <a
              routerLink="/login"
              style="font-family:var(--font-body);font-size:13px;font-weight:600;color:var(--color-primary);text-decoration:none;transition:color 150ms;"
              onmouseenter="this.style.textDecoration='underline'"
              onmouseleave="this.style.textDecoration='none'"
            >
              &larr; Kembali ke Login
            </a>
          </div>
        }
      </div>
    </div>
  `,
})
export class SetPasswordComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private authService = inject(AuthService);
  private fb = inject(FormBuilder);

  readonly token = signal<string>('');
  readonly hasTokenError = signal<boolean>(false);
  readonly isLoading = signal<boolean>(false);
  readonly isSubmitted = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  readonly resetForm = this.fb.group(
    {
      passwordBaru: ['', [Validators.required, Validators.minLength(8)]],
      konfirmasiPassword: ['', [Validators.required]],
    },
    { validators: passwordsMatchValidator }
  );

  ngOnInit(): void {
    const tokenParam = this.route.snapshot.queryParams['token'];
    if (!tokenParam || !tokenParam.trim()) {
      this.hasTokenError.set(true);
    } else {
      this.token.set(tokenParam.trim());
    }
  }

  get passwordControl() {
    return this.resetForm.controls.passwordBaru;
  }

  get konfirmasiControl() {
    return this.resetForm.controls.konfirmasiPassword;
  }

  onSubmit(): void {
    if (this.hasTokenError()) return;

    if (this.resetForm.invalid) {
      this.resetForm.markAllAsTouched();
      return;
    }

    const tokenVal = this.token();
    const passwordBaruVal = this.resetForm.value.passwordBaru || '';
    if (!tokenVal || !passwordBaruVal) return;

    this.isLoading.set(true);
    this.errorMessage.set(null);
    this.resetForm.disable();

    this.authService.resetPassword(tokenVal, passwordBaruVal).subscribe({
      next: () => {
        this.isLoading.set(false);
        this.isSubmitted.set(true);
      },
      error: (err: any) => {
        this.isLoading.set(false);
        this.resetForm.enable();

        const parsedEnvelope: ErrorEnvelope | null = err?.error?.error ? err.error : (err as ErrorEnvelope);
        const code = parsedEnvelope?.error?.code || err?.code;

        if (code === 'INVALID_TOKEN' || err?.status === 400 && err?.error?.error?.code === 'INVALID_TOKEN') {
          // Token expired, consumed, or invalid -> render Token Error State card (NOT Toast)
          this.hasTokenError.set(true);
        } else {
          // Technical error (Network/500/Timeout) -> render Toast notification
          let message = 'Gagal menghubungi server. Silakan periksa koneksi internet Anda dan coba lagi.';
          if (parsedEnvelope?.error?.message) {
            message = parsedEnvelope.error.message;
          } else if (err?.message) {
            message = err.message;
          }
          this.errorMessage.set(message);
        }
      },
    });
  }

  onDismissToast(): void {
    this.errorMessage.set(null);
  }
}
