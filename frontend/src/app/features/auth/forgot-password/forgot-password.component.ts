import {
  ChangeDetectionStrategy,
  Component,
  inject,
  signal,
} from '@angular/core';
import {
  FormBuilder,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { ToastComponent } from '../../../shared/components/toast/toast.component';

@Component({
  selector: 'app-forgot-password',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, RouterLink, ToastComponent],
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
        background-color: #F0FDFA;
        background-image: radial-gradient(ellipse 90% 55% at 50% 0%, rgba(8,145,178,0.10) 0%, transparent 65%);
      "
    >
      <!-- ── Brand Header ── -->
      <div class="flex flex-col items-center text-center mb-8">
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
        @if (!isSubmitted()) {
          <!-- Form Header -->
          <h1 style="font-family:var(--font-heading); font-size:20px; font-weight:700; color:#0F172A; margin-bottom:4px;">
            Lupa Password
          </h1>
          <p style="font-family:var(--font-body); font-size:13px; color:#64748B; line-height:1.5; margin-bottom:24px;">
            Masukkan email terdaftar Anda. Kami akan mengirimkan instruksi untuk mengatur ulang password.
          </p>

          <!-- Form Email -->
          <form [formGroup]="forgotForm" (ngSubmit)="onSubmit()" novalidate style="display:flex;flex-direction:column;gap:16px;">
            <div style="display:flex;flex-direction:column;gap:5px;">
              <label for="forgot-email" style="font-family:var(--font-body);font-size:12.5px;font-weight:600;color:#334155;">
                Email Terdaftar
              </label>
              <input
                id="forgot-email"
                type="email"
                formControlName="email"
                placeholder="contoh: nama@klinik.com"
                class="kl-input"
                autocomplete="email"
                [attr.aria-invalid]="emailControl.touched && emailControl.invalid ? 'true' : null"
              />
              @if (emailControl.touched && emailControl.errors?.['required']) {
                <span style="font-size:11.5px;color:#DC2626;" role="alert">Email wajib diisi</span>
              }
              @if (emailControl.touched && emailControl.errors?.['email']) {
                <span style="font-size:11.5px;color:#DC2626;" role="alert">Format email tidak valid</span>
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
              Kirim Link Reset
            </button>
          </form>
        } @else {
          <!-- Success State Card (Respon Generik 200) -->
          <div class="flex flex-col items-center text-center space-y-4">
            <div
              class="h-12 w-12 rounded-full bg-[#F0FDF4] border border-[#BBF7D0] flex items-center justify-center text-[var(--color-accent)]"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m5 12 5 5L20 7"/>
              </svg>
            </div>

            <div>
              <h2 style="font-family:var(--font-heading); font-size:18px; font-weight:700; color:#0F172A;">
                Instruksi Dikirim
              </h2>
              <p style="font-family:var(--font-body); font-size:13px; color:#475569; line-height:1.6; margin-top:8px;">
                Jika email <strong class="text-[var(--color-foreground)]">{{ submittedEmail() }}</strong> terdaftar di sistem kami, instruksi dan link reset password telah dikirimkan.
              </p>
            </div>

            <div class="p-3 bg-[var(--color-muted)] border border-[var(--color-border)] rounded-[var(--radius-md)] text-left w-full">
              <p style="font-family:var(--font-body); font-size:12px; color:#64748B; line-height:1.5;">
                 Link reset password berlaku selama <strong>1 jam</strong>. Silakan periksa folder Inbox atau Spam email Anda.
              </p>
            </div>

            <a
              routerLink="/login"
              class="kl-btn-primary w-full inline-flex justify-center items-center gap-2"
              style="text-decoration:none; margin-top:8px;"
            >
              Kembali ke Halaman Login
            </a>
          </div>
        }

        <!-- Back to login link (on form state) -->
        @if (!isSubmitted()) {
          <div style="text-align:center; margin-top:20px;">
            <a
              routerLink="/login"
              style="font-family:var(--font-body);font-size:13px;font-weight:600;color:#0891B2;text-decoration:none;transition:color 150ms;"
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
export class ForgotPasswordComponent {
  private authService = inject(AuthService);
  private fb = inject(FormBuilder);

  readonly isLoading = signal<boolean>(false);
  readonly isSubmitted = signal<boolean>(false);
  readonly submittedEmail = signal<string>('');
  readonly errorMessage = signal<string | null>(null);

  readonly forgotForm = this.fb.group({
    email: ['', [Validators.required, Validators.email]],
  });

  get emailControl() {
    return this.forgotForm.controls.email;
  }

  onSubmit(): void {
    if (this.forgotForm.invalid) {
      this.forgotForm.markAllAsTouched();
      return;
    }

    const emailValue = this.forgotForm.value.email?.trim() || '';
    if (!emailValue) return;

    this.isLoading.set(true);
    this.errorMessage.set(null);
    this.forgotForm.disable();

    this.authService.forgotPassword(emailValue).subscribe({
      next: () => {
        this.isLoading.set(false);
        this.submittedEmail.set(emailValue);
        this.isSubmitted.set(true);
      },
      error: (err: any) => {
        this.isLoading.set(false);
        this.forgotForm.enable();
        let message = 'Gagal menghubungi server. Silakan periksa koneksi internet Anda dan coba lagi.';
        if (err?.error?.message) {
          message = err.error.message;
        } else if (err?.message) {
          message = err.message;
        }
        this.errorMessage.set(message);
      },
    });
  }

  onDismissToast(): void {
    this.errorMessage.set(null);
  }
}
