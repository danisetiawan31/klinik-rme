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
import { toast } from '@spartan-ng/brain/sonner';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucideCheck, lucideLoader2, lucidePlus } from '@ng-icons/lucide';
import { HlmButton } from '../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../../shared/ui/input/src/lib/hlm-input';
import { HlmLabel } from '../../../shared/ui/label/src/lib/hlm-label';
import { AuthService } from '../../../core/auth/auth.service';

@Component({
  selector: 'app-forgot-password',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    RouterLink,
    HlmButton,
    NgIcon,
    HlmIconDirective,
    HlmInput,
    HlmLabel,
    ...HlmCardImports,
  ],
  providers: [provideIcons({ lucidePlus, lucideLoader2, lucideCheck })],
  templateUrl: './forgot-password.component.html',
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
        toast.error(message);
      },
    });
  }

  onDismissToast(): void {
    this.errorMessage.set(null);
  }
}
