import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideBarChart3,
  lucideFileText,
  lucideListOrdered,
  lucideSettings,
  lucideShieldCheck,
  lucideUserCheck,
  lucideUsers,
} from '@ng-icons/lucide';
import { AuthService } from '../../../core/auth/auth.service';
import { HlmIconDirective } from '../../../shared/ui/icon/src/lib/hlm-icon.directive';

export interface NavShortcut {
  label: string;
  route: string;
  description: string;
  iconName: string;
}

@Component({
  selector: 'app-landing',
  standalone: true,
  imports: [RouterLink, NgIcon, HlmIconDirective],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    provideIcons({
      lucideUsers,
      lucideListOrdered,
      lucideFileText,
      lucideUserCheck,
      lucideShieldCheck,
      lucideSettings,
      lucideBarChart3,
    }),
  ],
  templateUrl: './landing.component.html',
})
export class LandingComponent {
  private authService = inject(AuthService);

  readonly currentUser = this.authService.currentUser;
  readonly userName = computed(() => this.currentUser()?.nama || 'Pengguna');
  readonly userRoles = computed(() => this.currentUser()?.roles || []);

  readonly shortcuts = computed<NavShortcut[]>(() => {
    const roles = this.userRoles();
    const result: NavShortcut[] = [];

    if (roles.includes('petugas')) {
      result.push(
        { label: 'Pendaftaran Pasien', route: '/pasien', description: 'Registrasi & pencarian data pasien', iconName: 'lucideUsers' },
        { label: 'Kelola Antrian', route: '/antrian', description: 'Panggilan & penataan antrian pasien', iconName: 'lucideListOrdered' },
        { label: 'Laporan Harian', route: '/laporan-harian', description: 'Rekapitulasi kunjungan harian klinik', iconName: 'lucideBarChart3' }
      );
    }

    if (roles.includes('dokter')) {
      if (!result.some((r) => r.route === '/antrian')) {
        result.push({ label: 'Antrian Pasien', route: '/antrian', description: 'Daftar panggilan pasien masuk', iconName: 'lucideListOrdered' });
      }
      result.push(
        { label: 'Rekam Medis', route: '/rekam-medis', description: 'Pemeriksaan & pengisian rekam medis', iconName: 'lucideFileText' },
        { label: 'Riwayat Pasien', route: '/pasien/riwayat', description: 'Pencarian histori medis pasien', iconName: 'lucideUsers' }
      );
      if (!result.some((r) => r.route === '/laporan-harian')) {
        result.push({ label: 'Laporan Harian', route: '/laporan-harian', description: 'Laporan aktivitas konsultasi harian', iconName: 'lucideBarChart3' });
      }
    }

    if (roles.includes('admin')) {
      if (!result.some((r) => r.route === '/pasien')) {
        result.push({ label: 'Data Pasien', route: '/pasien', description: 'Kelola data seluruh pasien', iconName: 'lucideUsers' });
      }
      if (!result.some((r) => r.route === '/antrian')) {
        result.push({ label: 'Daftar Antrian', route: '/antrian', description: 'Pantau antrian & status tidak hadir', iconName: 'lucideListOrdered' });
      }
      result.push(
        { label: 'Manajemen Staff', route: '/admin/users', description: 'Kelola akun user & hak akses role', iconName: 'lucideUserCheck' },
        { label: 'Audit Log System', route: '/admin/audit-log', description: 'Log histori keamanan & rekam medis', iconName: 'lucideShieldCheck' },
        { label: 'Pengaturan Klinik', route: '/admin/pengaturan', description: 'Konfigurasi profil & jam operasional', iconName: 'lucideSettings' }
      );
      if (!result.some((r) => r.route === '/laporan-harian')) {
        result.push({ label: 'Laporan Harian', route: '/laporan-harian', description: 'Laporan ringkasan operasional', iconName: 'lucideBarChart3' });
      }
    }

    return result;
  });
}
