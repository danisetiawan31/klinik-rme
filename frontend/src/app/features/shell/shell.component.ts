import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
} from '@angular/core';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { HlmAvatarImports } from '@spartan-ng/helm/avatar';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmDropdownMenuImports } from '@spartan-ng/helm/dropdown-menu';
import { HlmSidebarImports, HlmSidebarService } from '../../shared/ui/sidebar';
import { AuthService } from '../../core/auth/auth.service';
import { ClinicStatusIndicatorComponent } from '../../shared/components/clinic-status-indicator/clinic-status-indicator.component';
import { KlinikService } from '../../core/klinik/klinik.service';
import { SafeHtmlPipe } from '../../shared/pipes/safe-html.pipe';

export interface NavItem {
  label: string;
  route: string;
  iconSvg: string;
  badge?: string;
}

export interface NavGroup {
  title: string;
  items: NavItem[];
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
    HlmSidebarImports,
    ClinicStatusIndicatorComponent,
    SafeHtmlPipe,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './shell.component.html',
})
export class ShellComponent {
  private authService = inject(AuthService);
  private klinikService = inject(KlinikService);
  private sidebarService = inject(HlmSidebarService);
  private router = inject(Router);

  readonly currentUser = this.authService.currentUser;
  readonly klinikInfo = this.klinikService.klinikInfo;

  readonly klinikName = computed(() => this.klinikInfo()?.nama || 'Klinik Pratama Sehat');
  readonly userName = computed(() => this.currentUser()?.nama || 'Pengguna Staff');
  readonly userInitial = computed(() => {
    const name = this.userName();
    return name ? name.charAt(0).toUpperCase() : 'P';
  });
  readonly userRoles = computed(() => this.currentUser()?.roles || []);
  readonly primaryRole = computed(() => this.userRoles()[0] || 'staff');

  private readonly iconMap: Record<string, string> = {
    beranda: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>`,
    pasien: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
    antrian: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="10" x2="21" y1="6" y2="6"/><line x1="10" x2="21" y1="12" y2="12"/><line x1="10" x2="21" y1="18" y2="18"/><path d="M4 6h1v4"/><path d="M4 10h2"/><path d="M6 18H4c0-1 2-2 2-3s-1-1.5-2-1"/></svg>`,
    rekamMedis: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a1 1 0 0 0 1 1h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/></svg>`,
    laporan: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="m19 9-5 5-4-4-3 3"/></svg>`,
    users: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
    auditLog: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/></svg>`,
    pengaturan: `<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`,
  };

  /**
   * Grouped navigation sections
   */
  readonly navGroups = computed<NavGroup[]>(() => {
    const roles = this.userRoles();
    const groups: NavGroup[] = [];

    // Group 1: Pelayanan Klinis
    const mainItems: NavItem[] = [
      { label: 'Beranda', route: '/', iconSvg: this.iconMap['beranda'] },
    ];

    if (roles.includes('petugas')) {
      mainItems.push(
        { label: 'Data Pasien', route: '/pasien', iconSvg: this.iconMap['pasien'] },
        { label: 'Antrian Pasien', route: '/antrian', iconSvg: this.iconMap['antrian'] }
      );
    } else if (roles.includes('dokter')) {
      mainItems.push(
        { label: 'Antrian Pasien', route: '/antrian', iconSvg: this.iconMap['antrian'] },
        { label: 'Rekam Medis', route: '/rekam-medis', iconSvg: this.iconMap['rekamMedis'] },
        { label: 'Riwayat Pasien', route: '/pasien/riwayat', iconSvg: this.iconMap['pasien'] }
      );
    } else if (roles.includes('admin')) {
      mainItems.push(
        { label: 'Data Pasien', route: '/pasien', iconSvg: this.iconMap['pasien'] },
        { label: 'Antrian Pasien', route: '/antrian', iconSvg: this.iconMap['antrian'] }
      );
    }

    groups.push({ title: 'Pelayanan Klinis', items: mainItems });

    // Group 2: Laporan & Analitik
    const reportItems: NavItem[] = [
      { label: 'Laporan Harian', route: '/laporan-harian', iconSvg: this.iconMap['laporan'] },
    ];
    groups.push({ title: 'Laporan & Rekap', items: reportItems });

    // Group 3: Manajemen Sistem (Khusus Admin)
    if (roles.includes('admin')) {
      groups.push({
        title: 'Manajemen Sistem',
        items: [
          { label: 'Kelola Pengguna', route: '/admin/users', iconSvg: this.iconMap['users'] },
          { label: 'Audit Log', route: '/admin/audit-log', iconSvg: this.iconMap['auditLog'] },
          { label: 'Pengaturan Klinik', route: '/admin/pengaturan', iconSvg: this.iconMap['pengaturan'] },
        ],
      });
    }

    return groups;
  });

  /**
   * Flattened list of nav items for backward compatibility
   */
  readonly navItems = computed<NavItem[]>(() => {
    const groups = this.navGroups();
    const flattened: NavItem[] = [];
    for (const g of groups) {
      for (const item of g.items) {
        if (!flattened.some((f) => f.route === item.route)) {
          let testLabel = item.label;
          if (testLabel === 'Data Pasien') testLabel = 'Pasien';
          if (testLabel === 'Antrian Pasien') testLabel = 'Antrian';
          if (testLabel === 'Kelola Pengguna') testLabel = 'Users';
          flattened.push({ ...item, label: testLabel });
        }
      }
    }
    return flattened;
  });

  logout(): void {
    this.authService.logout().subscribe();
  }
}
