import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
} from '@angular/core';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideBarChart3,
  lucideChevronsUpDown,
  lucideFileText,
  lucideHeartPulse,
  lucideLayoutDashboard,
  lucideListOrdered,
  lucideLogOut,
  lucideSettings,
  lucideShieldCheck,
  lucideUserCheck,
  lucideUsers,
} from '@ng-icons/lucide';
import { HlmAvatarImports } from '@spartan-ng/helm/avatar';
import { HlmDropdownMenuImports } from '@spartan-ng/helm/dropdown-menu';
import { AuthService } from '../../core/auth/auth.service';
import { KlinikService } from '../../core/klinik/klinik.service';
import { ClinicStatusIndicatorComponent } from '../../shared/components/clinic-status-indicator/clinic-status-indicator.component';
import { HlmButton } from '../../shared/ui/button/src/lib/hlm-button';
import { HlmIconDirective } from '../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmSidebarImports, HlmSidebarService } from '../../shared/ui/sidebar';

export interface NavItem {
  label: string;
  route: string;
  iconName: string;
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
    NgIcon,
    HlmIconDirective,
    HlmAvatarImports,
    HlmDropdownMenuImports,
    HlmSidebarImports,
    ClinicStatusIndicatorComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    provideIcons({
      lucideHeartPulse,
      lucideLayoutDashboard,
      lucideUsers,
      lucideListOrdered,
      lucideFileText,
      lucideBarChart3,
      lucideUserCheck,
      lucideShieldCheck,
      lucideSettings,
      lucideChevronsUpDown,
      lucideLogOut,
    }),
  ],
  templateUrl: './shell.component.html',
})
export class ShellComponent {
  private authService = inject(AuthService);
  private klinikService = inject(KlinikService);
  private sidebarService = inject(HlmSidebarService);
  private router = inject(Router);

  readonly currentUser = this.authService.currentUser;
  readonly klinikInfo = this.klinikService.klinikInfo;

  readonly klinikName = computed(() => this.klinikInfo()?.nama || 'Klinik Sehat Jaya');
  readonly userName = computed(() => this.currentUser()?.nama || 'Pengguna Staff');
  readonly userInitial = computed(() => {
    const name = this.userName();
    return name ? name.charAt(0).toUpperCase() : 'P';
  });
  readonly userRoles = computed(() => this.currentUser()?.roles || []);
  readonly primaryRole = computed(() => this.userRoles()[0] || 'staff');

  /**
   * Grouped navigation sections
   */
  readonly navGroups = computed<NavGroup[]>(() => {
    const roles = this.userRoles();
    const groups: NavGroup[] = [];

    // Group 1: Pelayanan Klinis
    const mainItems: NavItem[] = [
      { label: 'Beranda', route: '/', iconName: 'lucideLayoutDashboard' },
    ];

    if (roles.includes('petugas')) {
      mainItems.push(
        { label: 'Data Pasien', route: '/pasien', iconName: 'lucideUsers' },
        { label: 'Antrian Pasien', route: '/antrian', iconName: 'lucideListOrdered' }
      );
    } else if (roles.includes('dokter')) {
      mainItems.push(
        { label: 'Antrian Pasien', route: '/antrian', iconName: 'lucideListOrdered' },
        { label: 'Rekam Medis', route: '/rekam-medis', iconName: 'lucideFileText' },
        { label: 'Riwayat Pasien', route: '/pasien/riwayat', iconName: 'lucideUsers' }
      );
    } else if (roles.includes('admin')) {
      mainItems.push(
        { label: 'Data Pasien', route: '/pasien', iconName: 'lucideUsers' },
        { label: 'Antrian Pasien', route: '/antrian', iconName: 'lucideListOrdered' }
      );
    }

    groups.push({ title: 'Pelayanan Klinis', items: mainItems });

    // Group 2: Laporan & Analitik
    const reportItems: NavItem[] = [
      { label: 'Laporan Harian', route: '/laporan-harian', iconName: 'lucideBarChart3' },
    ];
    groups.push({ title: 'Laporan & Rekap', items: reportItems });

    // Group 3: Manajemen Sistem (Khusus Admin)
    if (roles.includes('admin')) {
      groups.push({
        title: 'Manajemen Sistem',
        items: [
          { label: 'Kelola Pengguna', route: '/admin/users', iconName: 'lucideUserCheck' },
          { label: 'Audit Log', route: '/admin/audit-log', iconName: 'lucideShieldCheck' },
          { label: 'Pengaturan Klinik', route: '/admin/pengaturan', iconName: 'lucideSettings' },
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
