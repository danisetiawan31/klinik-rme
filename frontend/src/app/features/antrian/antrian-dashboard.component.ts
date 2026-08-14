import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-antrian-dashboard',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './antrian-dashboard.component.html',
})
export class AntrianDashboardComponent {
  readonly authService = inject(AuthService);
}
