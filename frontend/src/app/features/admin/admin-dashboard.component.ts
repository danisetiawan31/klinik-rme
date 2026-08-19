import {
  ChangeDetectionStrategy,
  Component,
  inject,
  signal,
} from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideActivity,
  lucideFileText,
  lucideKey,
  lucideShield,
  lucideUsers,
} from '@ng-icons/lucide';
import { AuthService } from '../../core/auth/auth.service';
import { HlmButton } from '../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../shared/ui/icon/src/lib/hlm-icon.directive';
import { AdminAuditLogComponent } from './components/admin-audit-log/admin-audit-log.component';
import { AdminUsersComponent } from './components/admin-users/admin-users.component';

export type AdminTab = 'users' | 'audit-log' | 'klinik';

@Component({
  selector: 'app-admin-dashboard',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    NgIcon,
    HlmIconDirective,
    ...HlmCardImports,
    AdminUsersComponent,
    AdminAuditLogComponent,
  ],
  providers: [
    provideIcons({
      lucideUsers,
      lucideFileText,
      lucideKey,
      lucideShield,
      lucideActivity,
    }),
  ],
  templateUrl: './admin-dashboard.component.html',
})
export class AdminDashboardComponent {
  readonly authService = inject(AuthService);
  readonly activeTab = signal<AdminTab>('users');

  setTab(tab: AdminTab): void {
    this.activeTab.set(tab);
  }
}
