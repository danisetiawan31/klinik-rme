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
import { AuthService } from '../../../core/auth/auth.service';
import { SensitiveValueComponent } from '../../../shared/components/sensitive-value/sensitive-value.component';

@Component({
  selector: 'app-login',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, RouterLink, SensitiveValueComponent],
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
