import {
  ChangeDetectionStrategy,
  Component,
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
import { toast } from '@spartan-ng/brain/sonner';
import { HlmCardImports } from '../../shared/ui/card/src/index';
import { AuthService } from '../../core/auth/auth.service';
import { ErrorEnvelope } from '../../core/types/api-response.type';
import { SensitiveValueComponent } from '../../shared/components/sensitive-value/sensitive-value.component';

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
  selector: 'app-profil',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, SensitiveValueComponent, ...HlmCardImports],
  templateUrl: './profil.component.html',
})
export class ProfilComponent {
  readonly authService = inject(AuthService);
  private fb = inject(FormBuilder);

  readonly currentUser = this.authService.currentUser;
  readonly userRoles = () => this.currentUser()?.roles || [];

  readonly isLoading = signal<boolean>(false);
  readonly oldPasswordError = signal<string | null>(null);

  readonly toastMessage = signal<string | null>(null);
  readonly toastType = signal<'success' | 'error'>('success');

  readonly changePasswordForm = this.fb.group(
    {
      passwordLama: ['', [Validators.required]],
      passwordBaru: ['', [Validators.required, Validators.minLength(8)]],
      konfirmasiPassword: ['', [Validators.required]],
    },
    { validators: passwordsMatchValidator }
  );

  get oldPasswordControl() {
    return this.changePasswordForm.controls.passwordLama;
  }

  get newPasswordControl() {
    return this.changePasswordForm.controls.passwordBaru;
  }

  get konfirmasiControl() {
    return this.changePasswordForm.controls.konfirmasiPassword;
  }

  onClearOldPasswordError(): void {
    this.oldPasswordError.set(null);
  }

  onDismissToast(): void {
    this.toastMessage.set(null);
  }

  onSubmit(): void {
    this.oldPasswordError.set(null);
    this.toastMessage.set(null);

    if (this.changePasswordForm.invalid) {
      this.changePasswordForm.markAllAsTouched();
      return;
    }

    const { passwordLama, passwordBaru } = this.changePasswordForm.value;
    if (!passwordLama || !passwordBaru) return;

    this.isLoading.set(true);
    this.changePasswordForm.disable();

    this.authService.changePassword(passwordLama, passwordBaru).subscribe({
      next: () => {
        this.isLoading.set(false);
        this.changePasswordForm.enable();
        this.changePasswordForm.reset();

        this.toastType.set('success');
        this.toastMessage.set('Password berhasil diubah');
        toast.success('Password berhasil diubah');
      },
      error: (err: any) => {
        this.isLoading.set(false);
        this.changePasswordForm.enable();

        const parsedEnvelope: ErrorEnvelope | null = err?.error?.error ? err.error : (err as ErrorEnvelope);
        const code = parsedEnvelope?.error?.code || err?.code;

        if (code === 'INVALID_PASSWORD' || (err?.status === 400 && err?.error?.error?.code === 'INVALID_PASSWORD')) {
          const msg = parsedEnvelope?.error?.message || 'Kata sandi saat ini salah';
          this.oldPasswordError.set(msg);
        } else {
          let message = 'Gagal memperbarui password di server';
          if (parsedEnvelope?.error?.message) {
            message = parsedEnvelope.error.message;
          } else if (err?.message) {
            message = err.message;
          }
          this.toastType.set('error');
          this.toastMessage.set(message);
          toast.error(message);
        }
      },
    });
  }
}
