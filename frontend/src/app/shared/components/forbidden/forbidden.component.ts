import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucideShieldX } from '@ng-icons/lucide';
import { HlmButton } from '../../ui/button/src/lib/hlm-button';
import { HlmIconDirective } from '../../ui/icon/src/lib/hlm-icon.directive';
import { AuthService } from '../../../core/auth/auth.service';

@Component({
  selector: 'app-forbidden',
  standalone: true,
  imports: [RouterLink, HlmButton, NgIcon, HlmIconDirective],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [provideIcons({ lucideShieldX })],
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
