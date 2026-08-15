import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { HlmButton } from '../../shared/ui/button/src/lib/hlm-button';
import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-admin-dashboard',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [HlmButton],
  templateUrl: './admin-dashboard.component.html',
})
export class AdminDashboardComponent {
  readonly authService = inject(AuthService);
}
