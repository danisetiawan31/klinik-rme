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
import { toast } from '@spartan-ng/brain/sonner';
import { HlmButton } from '../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../shared/ui/card/src/index';
import { AuthService } from '../../../core/auth/auth.service';
import { ErrorEnvelope } from '../../../core/types/api-response.type';
import { SensitiveValueComponent } from '../../../shared/components/sensitive-value/sensitive-value.component';

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
  imports: [ReactiveFormsModule, RouterLink, SensitiveValueComponent, HlmButton, ...HlmCardImports],
  templateUrl: './set-password.component.html',
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

        if (code === 'INVALID_TOKEN' || (err?.status === 400 && err?.error?.error?.code === 'INVALID_TOKEN')) {
          this.hasTokenError.set(true);
        } else {
          let message = 'Gagal menghubungi server. Silakan periksa koneksi internet Anda dan coba lagi.';
          if (parsedEnvelope?.error?.message) {
            message = parsedEnvelope.error.message;
          } else if (err?.message) {
            message = err.message;
          }
          this.errorMessage.set(message);
          toast.error(message);
        }
      },
    });
  }

  onDismissToast(): void {
    this.errorMessage.set(null);
  }
}
