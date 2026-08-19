import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { ActivatedRoute, NavigationEnd, Router } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideActivity,
  lucideFileText,
  lucideKey,
  lucideShield,
  lucideUsers,
} from '@ng-icons/lucide';
import { filter } from 'rxjs';
import { AuthService } from '../../core/auth/auth.service';
import { HlmButton } from '../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../shared/ui/icon/src/lib/hlm-icon.directive';
import { AdminAuditLogComponent } from './components/admin-audit-log/admin-audit-log.component';
import { AdminKlinikComponent } from './components/admin-klinik/admin-klinik.component';
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
    AdminKlinikComponent,
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
export class AdminDashboardComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);
  readonly authService = inject(AuthService);
  readonly activeTab = signal<AdminTab>('users');

  ngOnInit(): void {
    // 1. Initial resolution
    this.syncTabFromUrl(this.router.url, this.route.snapshot.paramMap.get('subtab'));

    // 2. React to route parameter changes when Angular reuses the component
    const paramSub = this.route.paramMap.subscribe((params) => {
      this.syncTabFromUrl(this.router.url, params.get('subtab'));
    });

    // 3. React to router NavigationEnd events for comprehensive URL sync
    const routerSub = this.router.events
      .pipe(filter((event) => event instanceof NavigationEnd))
      .subscribe((event: any) => {
        this.syncTabFromUrl(event.urlAfterRedirects || event.url);
      });

    this.destroyRef.onDestroy(() => {
      paramSub.unsubscribe();
      routerSub.unsubscribe();
    });
  }

  private syncTabFromUrl(url: string, subtabParam?: string | null): void {
    const subtab = subtabParam || '';
    if (subtab === 'audit-log' || url.includes('audit-log')) {
      this.activeTab.set('audit-log');
    } else if (
      subtab === 'klinik' ||
      subtab === 'pengaturan' ||
      url.includes('pengaturan') ||
      url.includes('klinik')
    ) {
      this.activeTab.set('klinik');
    } else {
      this.activeTab.set('users');
    }
  }

  setTab(tab: AdminTab): void {
    this.activeTab.set(tab);
    const targetRoute = tab === 'klinik' ? 'pengaturan' : tab;
    this.router.navigate(['/admin', targetRoute]);
  }
}
