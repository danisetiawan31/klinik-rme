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
import { toast } from '@spartan-ng/brain/sonner';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucideLoader2, lucidePlus } from '@ng-icons/lucide';
import { HlmButton } from '../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../../shared/ui/input/src/lib/hlm-input';
import { HlmLabel } from '../../../shared/ui/label/src/lib/hlm-label';
import { AuthService } from '../../../core/auth/auth.service';
import { SensitiveValueComponent } from '../../../shared/components/sensitive-value/sensitive-value.component';

@Component({
  selector: 'app-login',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    RouterLink,
    SensitiveValueComponent,
    HlmButton,
    NgIcon,
    HlmIconDirective,
    HlmInput,
    HlmLabel,
    ...HlmCardImports,
  ],
  providers: [provideIcons({ lucidePlus, lucideLoader2 })],
  templateUrl: './login.component.html',
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
        const err = this.authService.authError() || 'Email atau kata sandi tidak valid';
        toast.error(err);
      },
    });
  }
}
