import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';

import { SafeHtmlPipe } from '../../../shared/pipes/safe-html.pipe';

export interface NavShortcut {
  label: string;
  route: string;
  description: string;
  iconSvg: string;
}

@Component({
  selector: 'app-landing',
  standalone: true,
  imports: [RouterLink, SafeHtmlPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
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

    const iconMap: Record<string, string> = {
      pasien: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
      antrian: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="10" x2="21" y1="6" y2="6"/><line x1="10" x2="21" y1="12" y2="12"/><line x1="10" x2="21" y1="18" y2="18"/><path d="M4 6h1v4"/><path d="M4 10h2"/><path d="M6 18H4c0-1 2-2 2-3s-1-1.5-2-1"/></svg>`,
      rekamMedis: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a1 1 0 0 0 1 1h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/></svg>`,
      users: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
      auditLog: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/></svg>`,
      pengaturan: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`,
      laporan: `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="m19 9-5 5-4-4-3 3"/></svg>`,
    };

    if (roles.includes('petugas')) {
      result.push(
        { label: 'Pendaftaran Pasien', route: '/pasien', description: 'Registrasi & pencarian data pasien', iconSvg: iconMap['pasien'] },
        { label: 'Kelola Antrian', route: '/antrian', description: 'Panggilan & penataan antrian pasien', iconSvg: iconMap['antrian'] },
        { label: 'Laporan Harian', route: '/laporan-harian', description: 'Rekapitulasi kunjungan harian klinik', iconSvg: iconMap['laporan'] }
      );
    }

    if (roles.includes('dokter')) {
      if (!result.some((r) => r.route === '/antrian')) {
        result.push({ label: 'Antrian Pasien', route: '/antrian', description: 'Daftar panggilan pasien masuk', iconSvg: iconMap['antrian'] });
      }
      result.push(
        { label: 'Rekam Medis', route: '/rekam-medis', description: 'Pemeriksaan & pengisian rekam medis', iconSvg: iconMap['rekamMedis'] },
        { label: 'Riwayat Pasien', route: '/pasien/riwayat', description: 'Pencarian histori medis pasien', iconSvg: iconMap['pasien'] }
      );
      if (!result.some((r) => r.route === '/laporan-harian')) {
        result.push({ label: 'Laporan Harian', route: '/laporan-harian', description: 'Laporan aktivitas konsultasi harian', iconSvg: iconMap['laporan'] });
      }
    }

    if (roles.includes('admin')) {
      if (!result.some((r) => r.route === '/pasien')) {
        result.push({ label: 'Data Pasien', route: '/pasien', description: 'Kelola data seluruh pasien', iconSvg: iconMap['pasien'] });
      }
      if (!result.some((r) => r.route === '/antrian')) {
        result.push({ label: 'Daftar Antrian', route: '/antrian', description: 'Pantau antrian & status tidak hadir', iconSvg: iconMap['antrian'] });
      }
      result.push(
        { label: 'Manajemen Staff', route: '/admin/users', description: 'Kelola akun user & hak akses role', iconSvg: iconMap['users'] },
        { label: 'Audit Log System', route: '/admin/audit-log', description: 'Log histori keamanan & rekam medis', iconSvg: iconMap['auditLog'] },
        { label: 'Pengaturan Klinik', route: '/admin/pengaturan', description: 'Konfigurasi profil & jam operasional', iconSvg: iconMap['pengaturan'] }
      );
      if (!result.some((r) => r.route === '/laporan-harian')) {
        result.push({ label: 'Laporan Harian', route: '/laporan-harian', description: 'Laporan ringkasan operasional', iconSvg: iconMap['laporan'] });
      }
    }

    return result;
  });
}
