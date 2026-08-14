import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  signal,
} from '@angular/core';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { HlmAvatarImports } from '@spartan-ng/helm/avatar';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmDropdownMenuImports } from '@spartan-ng/helm/dropdown-menu';
import { HlmSheetImports } from '@spartan-ng/helm/sheet';
import { AuthService } from '../../core/auth/auth.service';
import { ClinicStatusIndicatorComponent } from '../../shared/components/clinic-status-indicator/clinic-status-indicator.component';

export interface NavItem {
  label: string;
  route: string;
  iconSvg: string;
}

@Component({
  selector: 'app-shell',
  standalone: true,
  imports: [
    RouterOutlet,
    RouterLink,
    RouterLinkActive,
    HlmButton,
    HlmAvatarImports,
    HlmDropdownMenuImports,
    HlmSheetImports,
    ClinicStatusIndicatorComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="min-h-screen bg-[var(--color-background)] font-sans flex flex-col">
      <!-- Top Navigation Header -->
      <header
        class="h-16 bg-[var(--color-card)] border-b border-[var(--color-border)] px-4 sm:px-6 flex items-center justify-between sticky top-0 z-30 shadow-[var(--shadow-1)]"
      >
        <div class="flex items-center gap-3">
          <!-- Mobile Drawer Trigger (< lg) -->
          <hlm-sheet side="left">
            <button
              hlmBtn
              variant="ghost"
              size="icon"
              hlmSheetTrigger
              class="lg:hidden text-[var(--color-foreground)] cursor-pointer"
              aria-label="Buka Menu Sidebar"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="4" x2="20" y1="12" y2="12"/>
                <line x1="4" x2="20" y1="6" y2="6"/>
                <line x1="4" x2="20" y1="18" y2="18"/>
              </svg>
            </button>
            <hlm-sheet-content class="w-72 p-0 bg-[var(--color-card)] border-r border-[var(--color-border)]">
              <div class="p-6 border-b border-[var(--color-border)] flex items-center gap-3">
                <div class="h-9 w-9 rounded-lg bg-[var(--color-primary)] text-[var(--color-primary-foreground)] flex items-center justify-center font-bold text-lg">
                  K
                </div>
                <div>
                  <h2 class="font-heading font-bold text-base text-[var(--color-foreground)]">
                    Klinik RME
                  </h2>
                  <p class="text-xs text-[var(--color-muted-foreground)]">Sistem Staff Klinik</p>
                </div>
              </div>
              <nav class="p-4 space-y-1">
                @for (item of navItems(); track item.route) {
                  <a
                    [routerLink]="item.route"
                    routerLinkActive="bg-[var(--color-primary)] text-[var(--color-primary-foreground)]"
                    [routerLinkActiveOptions]="{ exact: item.route === '/' }"
                    class="flex items-center gap-3 px-3 py-2.5 rounded-[var(--radius-md)] text-sm font-medium text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
                  >
                    <span [innerHTML]="item.iconSvg" class="flex items-center"></span>
                    <span>{{ item.label }}</span>
                  </a>
                }
              </nav>
            </hlm-sheet-content>
          </hlm-sheet>

          <!-- Desktop Sidebar Toggle Toggle (lg+) -->
          <button
            hlmBtn
            variant="ghost"
            size="icon"
            (click)="toggleDesktopSidebar()"
            class="hidden lg:flex text-[var(--color-foreground)] cursor-pointer"
            aria-label="Toggle Sidebar Desktop"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect width="18" height="18" x="3" y="3" rx="2"/>
              <path d="M9 3v18"/>
            </svg>
          </button>

          <!-- Brand Logo -->
          <a routerLink="/" class="flex items-center gap-2.5">
            <div class="h-8 w-8 rounded-md bg-[var(--color-primary)] text-[var(--color-primary-foreground)] flex items-center justify-center font-bold text-base shadow-sm">
              K
            </div>
            <span class="font-heading text-lg font-bold text-[var(--color-foreground)] hidden sm:inline-block tracking-tight">
              Klinik RME
            </span>
          </a>
        </div>

        <!-- Header Controls (Status & User Menu) -->
        <div class="flex items-center gap-3 sm:gap-4">
          <!-- Clinic Status Badge -->
          <app-clinic-status-indicator />

          <!-- User Menu Dropdown -->
          <div class="relative">
            <button
              hlmBtn
              variant="ghost"
              [hlmDropdownMenuTrigger]="userMenu"
              class="flex items-center gap-2 px-2 py-1.5 rounded-full hover:bg-[var(--color-muted)] transition-colors cursor-pointer"
              aria-label="Menu Pengguna"
            >
              <hlm-avatar size="sm" class="bg-[var(--color-primary)] text-[var(--color-primary-foreground)] font-semibold text-xs">
                <span hlmAvatarFallback>{{ userInitial() }}</span>
              </hlm-avatar>
              <span class="text-sm font-medium text-[var(--color-foreground)] hidden md:inline-block max-w-[120px] truncate">
                {{ userName() }}
              </span>
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-[var(--color-muted-foreground)]">
                <path d="m6 9 6 6 6-6"/>
              </svg>
            </button>

            <!-- Dropdown Menu Content -->
            <ng-template #userMenu>
              <div hlmDropdownMenu class="w-56 p-1 bg-[var(--color-card)] border border-[var(--color-border)] rounded-[var(--radius-md)] shadow-[var(--shadow-3)]">
                <div class="px-3 py-2.5 border-b border-[var(--color-border)]">
                  <p class="text-xs text-[var(--color-muted-foreground)]">Masuk sebagai</p>
                  <p class="text-sm font-semibold text-[var(--color-foreground)] truncate mt-0.5">
                    {{ userName() }}
                  </p>
                  <p class="text-xs text-[var(--color-primary)] font-medium mt-0.5 capitalize">
                    {{ primaryRole() }}
                  </p>
                </div>
                <div class="py-1">
                  <a
                    routerLink="/profil"
                    hlmDropdownMenuItem
                    class="w-full flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-foreground)] hover:bg-[var(--color-muted)] rounded-sm cursor-pointer transition-colors"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
                      <circle cx="12" cy="12" r="3"/>
                    </svg>
                    Pengaturan Akun
                  </a>
                  <button
                    hlmDropdownMenuItem
                    (click)="logout()"
                    class="w-full flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-destructive)] hover:bg-red-50 rounded-sm cursor-pointer transition-colors"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                      <polyline points="16 17 21 12 16 7"/>
                      <line x1="21" x2="9" y1="12" y2="12"/>
                    </svg>
                    Keluar (Logout)
                  </button>
                </div>
              </div>
            </ng-template>
          </div>
        </div>
      </header>

      <!-- Shell Content Body (Desktop Sidebar + Main Area) -->
      <div class="flex-1 flex overflow-hidden">
        <!-- Desktop Sidebar (lg+) -->
        <aside
          class="hidden lg:flex flex-col border-r border-[var(--color-border)] bg-[var(--color-card)] transition-all duration-300 z-20 shrink-0"
          [class.w-64]="!isSidebarCollapsed()"
          [class.w-20]="isSidebarCollapsed()"
        >
          <nav class="flex-1 p-3 space-y-1.5 overflow-y-auto">
            @for (item of navItems(); track item.route) {
              <a
                [routerLink]="item.route"
                routerLinkActive="bg-[var(--color-primary)] text-[var(--color-primary-foreground)]"
                [routerLinkActiveOptions]="{ exact: item.route === '/' }"
                class="flex items-center gap-3.5 px-3.5 py-2.5 rounded-[var(--radius-md)] text-sm font-medium text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-all group"
                [title]="isSidebarCollapsed() ? item.label : ''"
              >
                <span [innerHTML]="item.iconSvg" class="flex items-center shrink-0"></span>
                @if (!isSidebarCollapsed()) {
                  <span class="truncate">{{ item.label }}</span>
                }
              </a>
            }
          </nav>
        </aside>

        <!-- Main Content Workspace Area -->
        <main class="flex-1 overflow-y-auto p-4 sm:p-6 lg:p-8">
          <router-outlet />
        </main>
      </div>
    </div>
  `,
})
export class ShellComponent {
  private authService = inject(AuthService);
  private router = inject(Router);

  readonly isSidebarCollapsed = signal<boolean>(false);
  readonly currentUser = this.authService.currentUser;

  readonly userName = computed(() => this.currentUser()?.nama || 'Pengguna Staff');
  readonly userInitial = computed(() => {
    const name = this.userName();
    return name ? name.charAt(0).toUpperCase() : 'P';
  });
  readonly userRoles = computed(() => this.currentUser()?.roles || []);
  readonly primaryRole = computed(() => this.userRoles()[0] || 'staff');

  readonly navItems = computed<NavItem[]>(() => {
    const roles = this.userRoles();
    const items: NavItem[] = [
      {
        label: 'Beranda',
        route: '/',
        iconSvg: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>`,
      },
    ];

    const iconMap: Record<string, string> = {
      pasien: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
      antrian: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="10" x2="21" y1="6" y2="6"/><line x1="10" x2="21" y1="12" y2="12"/><line x1="10" x2="21" y1="18" y2="18"/><path d="M4 6h1v4"/><path d="M4 10h2"/><path d="M6 18H4c0-1 2-2 2-3s-1-1.5-2-1"/></svg>`,
      rekamMedis: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a1 1 0 0 0 1 1h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/></svg>`,
      users: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
      auditLog: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/></svg>`,
      pengaturan: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`,
      laporan: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="m19 9-5 5-4-4-3 3"/></svg>`,
    };

    if (roles.includes('petugas')) {
      items.push(
        { label: 'Pasien', route: '/pasien', iconSvg: iconMap['pasien'] },
        { label: 'Antrian', route: '/antrian', iconSvg: iconMap['antrian'] },
        { label: 'Laporan Harian', route: '/laporan-harian', iconSvg: iconMap['laporan'] }
      );
    } else if (roles.includes('dokter')) {
      items.push(
        { label: 'Antrian', route: '/antrian', iconSvg: iconMap['antrian'] },
        { label: 'Rekam Medis', route: '/rekam-medis', iconSvg: iconMap['rekamMedis'] },
        { label: 'Riwayat Pasien', route: '/pasien/riwayat', iconSvg: iconMap['pasien'] },
        { label: 'Laporan Harian', route: '/laporan-harian', iconSvg: iconMap['laporan'] }
      );
    } else if (roles.includes('admin')) {
      items.push(
        { label: 'Pasien', route: '/pasien', iconSvg: iconMap['pasien'] },
        { label: 'Antrian', route: '/antrian', iconSvg: iconMap['antrian'] },
        { label: 'Users', route: '/admin/users', iconSvg: iconMap['users'] },
        { label: 'Audit Log', route: '/admin/audit-log', iconSvg: iconMap['auditLog'] },
        { label: 'Pengaturan Klinik', route: '/admin/pengaturan', iconSvg: iconMap['pengaturan'] },
        { label: 'Laporan Harian', route: '/laporan-harian', iconSvg: iconMap['laporan'] }
      );
    }

    return items;
  });

  toggleDesktopSidebar(): void {
    this.isSidebarCollapsed.update((val) => !val);
  }

  logout(): void {
    this.authService.logout().subscribe();
  }
}
