import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { HlmButton } from '../../ui/button/src/lib/hlm-button';
import { AuthService } from '../../../core/auth/auth.service';

@Component({
  selector: 'app-forbidden',
  standalone: true,
  imports: [RouterLink, HlmButton],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './forbidden.component.html',
})
export class ForbiddenComponent {
  private authService = inject(AuthService);
  private router = inject(Router);

  goBack(): void {
    const landingRoute = this.authService.getLandingRoute();
    this.router.navigate([landingRoute]);
  }
}
