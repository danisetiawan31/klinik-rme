import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  inject,
  signal,
} from '@angular/core';
import { RouterLink } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideActivity,
  lucideAlertCircle,
  lucideAlertTriangle,
  lucideArrowRight,
  lucideBarChart3,
  lucideCalendar,
  lucideCalendarDays,
  lucideCheck,
  lucideCheckCircle2,
  lucideChevronRight,
  lucideClock,
  lucideFileText,
  lucideHeartPulse,
  lucideInbox,
  lucideListOrdered,
  lucideSettings,
  lucideShieldCheck,
  lucideSparkles,
  lucideTrendingUp,
  lucideUserCheck,
  lucideUserPlus,
  lucideUsers,
} from '@ng-icons/lucide';
import { AuthService } from '../../../core/auth/auth.service';
import { KlinikService } from '../../../core/klinik/klinik.service';
import {
  formatJakartaDayDate,
  getJakartaTimeString,
} from '../../../core/utils/date.utils';
import { AntrianService } from '../../antrian/antrian.service';
import { KunjunganListItem } from '../../antrian/antrian.types';
import { PriorityBadgeComponent } from '../../../shared/components/priority-badge/priority-badge.component';
import { StatusBadgeComponent } from '../../../shared/components/status-badge/status-badge.component';
import { HlmEmptyImports } from '../../../shared/ui/empty/src/lib/hlm-empty';
import { HlmIconDirective } from '../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmSkeletonImports } from '../../../shared/ui/skeleton/src/lib/hlm-skeleton';

export interface NavShortcut {
  label: string;
  route: string;
  description: string;
  iconName: string;
  badgeNumber: string;
  cardGradientClass: string;
  iconBgClass: string;
  iconBorderClass: string;
  iconColorClass: string;
  textColorClass: string;
  watermarkColorClass: string;
}

@Component({
  selector: 'app-landing',
  standalone: true,
  imports: [
    RouterLink,
    NgIcon,
    HlmIconDirective,
    StatusBadgeComponent,
    PriorityBadgeComponent,
    HlmSkeletonImports,
    HlmEmptyImports,
  ],
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
      lucideCalendar,
      lucideCalendarDays,
      lucideClock,
      lucideCheckCircle2,
      lucideAlertTriangle,
      lucideAlertCircle,
      lucideCheck,
      lucideHeartPulse,
      lucideSparkles,
      lucideTrendingUp,
      lucideArrowRight,
      lucideChevronRight,
      lucideInbox,
      lucideActivity,
      lucideUserPlus,
    }),
  ],
  templateUrl: './landing.component.html',
})
export class LandingComponent {
  private authService = inject(AuthService);
  private antrianService = inject(AntrianService);
  private klinikService = inject(KlinikService);
  private destroyRef = inject(DestroyRef);

  readonly currentUser = this.authService.currentUser;
  readonly userName = computed(() => this.currentUser()?.nama || 'Pengguna');
  readonly userRoles = computed(() => this.currentUser()?.roles || []);
  readonly isDokter = computed(() => this.userRoles().includes('dokter'));

  readonly klinikInfo = this.klinikService.klinikInfo;
  readonly isKlinikBuka = computed(() => this.klinikService.isKlinikBuka(this.klinikInfo()));
  readonly jamOperasionalStr = computed(() => {
    const info = this.klinikInfo();
    if (info?.jamBuka && info?.jamTutup) {
      return `Senin – Sabtu · ${info.jamBuka} – ${info.jamTutup} WIB`;
    }
    return 'Senin – Sabtu · 08:00 – 20:00 WIB';
  });

  readonly now = signal<Date>(new Date());
  readonly antrianList = signal<KunjunganListItem[]>([]);
  readonly isLoadingAntrian = signal<boolean>(false);

  // Live formatted clock & date
  readonly currentDateStr = computed(() => formatJakartaDayDate(this.now()));
  readonly currentTimeStr = computed(() => `${getJakartaTimeString(this.now())} WIB`);

  // Computed summary metrics
  readonly totalPasien = computed(() => this.antrianList().length);
  readonly antrianMenunggu = computed(
    () => this.antrianList().filter((a) => a.status === 'menunggu').length
  );
  readonly dipanggilCount = computed(
    () => this.antrianList().filter((a) => a.status === 'dipanggil').length
  );
  readonly selesaiDilayani = computed(
    () => this.antrianList().filter((a) => a.status === 'selesai').length
  );
  readonly pasienPrioritas = computed(
    () => this.antrianList().filter((a) => a.isPriority).length
  );

  readonly completionPercentage = computed(() => {
    const total = this.totalPasien();
    if (!total) return 0;
    return Math.round((this.selesaiDilayani() / total) * 100);
  });

  readonly recentAntrian = computed(() => this.antrianList().slice(0, 4));

  readonly shortcuts = computed<NavShortcut[]>(() => {
    const roles = this.userRoles();
    const result: NavShortcut[] = [];

    if (roles.includes('petugas')) {
      result.push(
        {
          label: 'Pendaftaran Pasien',
          route: '/pasien',
          description: 'Registrasi & pencarian data pasien',
          iconName: 'lucideUsers',
          badgeNumber: '01',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-cyan-500/10 hover:border-cyan-400/50',
          iconBgClass: 'bg-cyan-500/10',
          iconBorderClass: 'border-cyan-500/20',
          iconColorClass: 'text-cyan-600',
          textColorClass: 'text-cyan-600 group-hover:text-cyan-700',
          watermarkColorClass: 'text-cyan-600/30',
        },
        {
          label: 'Kelola Antrian',
          route: '/antrian',
          description: 'Panggilan & penataan antrian pasien',
          iconName: 'lucideListOrdered',
          badgeNumber: '02',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-sky-500/10 hover:border-sky-400/50',
          iconBgClass: 'bg-sky-500/10',
          iconBorderClass: 'border-sky-500/20',
          iconColorClass: 'text-sky-600',
          textColorClass: 'text-sky-600 group-hover:text-sky-700',
          watermarkColorClass: 'text-sky-600/30',
        },
        {
          label: 'Laporan Harian',
          route: '/laporan-harian',
          description: 'Rekapitulasi kunjungan harian klinik',
          iconName: 'lucideTrendingUp',
          badgeNumber: '03',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-emerald-500/10 hover:border-emerald-400/50',
          iconBgClass: 'bg-emerald-500/10',
          iconBorderClass: 'border-emerald-500/20',
          iconColorClass: 'text-emerald-600',
          textColorClass: 'text-emerald-600 group-hover:text-emerald-700',
          watermarkColorClass: 'text-emerald-600/30',
        }
      );
    }

    if (roles.includes('dokter')) {
      result.push(
        {
          label: 'Antrian Pasien',
          route: '/antrian',
          description: 'Kelola dan pantau antrian pasien masuk',
          iconName: 'lucideListOrdered',
          badgeNumber: '01',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-cyan-500/10 hover:border-cyan-400/50',
          iconBgClass: 'bg-cyan-500/10',
          iconBorderClass: 'border-cyan-500/20',
          iconColorClass: 'text-cyan-600',
          textColorClass: 'text-cyan-600 group-hover:text-cyan-700',
          watermarkColorClass: 'text-cyan-600/30',
        },
        {
          label: 'Rekam Medis',
          route: '/antrian',
          description: 'Pemeriksaan & pengisian rekam medis pasien',
          iconName: 'lucideFileText',
          badgeNumber: '02',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-blue-500/10 hover:border-blue-400/50',
          iconBgClass: 'bg-blue-500/10',
          iconBorderClass: 'border-blue-500/20',
          iconColorClass: 'text-blue-600',
          textColorClass: 'text-blue-600 group-hover:text-blue-700',
          watermarkColorClass: 'text-blue-600/30',
        },
        {
          label: 'Riwayat Pasien',
          route: '/pasien',
          description: 'Lihat dan cari histori medis pasien',
          iconName: 'lucideUsers',
          badgeNumber: '03',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-purple-500/10 hover:border-purple-400/50',
          iconBgClass: 'bg-purple-500/10',
          iconBorderClass: 'border-purple-500/20',
          iconColorClass: 'text-purple-600',
          textColorClass: 'text-purple-600 group-hover:text-purple-700',
          watermarkColorClass: 'text-purple-600/30',
        },
        {
          label: 'Laporan Harian',
          route: '/laporan-harian',
          description: 'Laporan aktivitas konsultasi harian',
          iconName: 'lucideTrendingUp',
          badgeNumber: '04',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-emerald-500/10 hover:border-emerald-400/50',
          iconBgClass: 'bg-emerald-500/10',
          iconBorderClass: 'border-emerald-500/20',
          iconColorClass: 'text-emerald-600',
          textColorClass: 'text-emerald-600 group-hover:text-emerald-700',
          watermarkColorClass: 'text-emerald-600/30',
        }
      );
    }

    if (roles.includes('admin')) {
      if (!result.some((r) => r.route === '/pasien')) {
        result.push({
          label: 'Data Pasien',
          route: '/pasien',
          description: 'Kelola data seluruh pasien',
          iconName: 'lucideUsers',
          badgeNumber: '01',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-cyan-500/10 hover:border-cyan-400/50',
          iconBgClass: 'bg-cyan-500/10',
          iconBorderClass: 'border-cyan-500/20',
          iconColorClass: 'text-cyan-600',
          textColorClass: 'text-cyan-600 group-hover:text-cyan-700',
          watermarkColorClass: 'text-cyan-600/30',
        });
      }
      if (!result.some((r) => r.route === '/antrian')) {
        result.push({
          label: 'Daftar Antrian',
          route: '/antrian',
          description: 'Pantau antrian & status tidak hadir',
          iconName: 'lucideListOrdered',
          badgeNumber: '02',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-sky-500/10 hover:border-sky-400/50',
          iconBgClass: 'bg-sky-500/10',
          iconBorderClass: 'border-sky-500/20',
          iconColorClass: 'text-sky-600',
          textColorClass: 'text-sky-600 group-hover:text-sky-700',
          watermarkColorClass: 'text-sky-600/30',
        });
      }
      result.push(
        {
          label: 'Manajemen Staff',
          route: '/admin/users',
          description: 'Kelola akun user & hak akses role',
          iconName: 'lucideUserCheck',
          badgeNumber: '03',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-purple-500/10 hover:border-purple-400/50',
          iconBgClass: 'bg-purple-500/10',
          iconBorderClass: 'border-purple-500/20',
          iconColorClass: 'text-purple-600',
          textColorClass: 'text-purple-600 group-hover:text-purple-700',
          watermarkColorClass: 'text-purple-600/30',
        },
        {
          label: 'Audit Log System',
          route: '/admin/audit-log',
          description: 'Log histori keamanan & rekam medis',
          iconName: 'lucideShieldCheck',
          badgeNumber: '04',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-amber-500/10 hover:border-amber-400/50',
          iconBgClass: 'bg-amber-500/10',
          iconBorderClass: 'border-amber-500/20',
          iconColorClass: 'text-amber-600',
          textColorClass: 'text-amber-600 group-hover:text-amber-700',
          watermarkColorClass: 'text-amber-600/30',
        },
        {
          label: 'Pengaturan Klinik',
          route: '/admin/pengaturan',
          description: 'Konfigurasi profil & jam operasional',
          iconName: 'lucideSettings',
          badgeNumber: '05',
          cardGradientClass: 'bg-gradient-to-br from-card via-card to-slate-500/10 hover:border-slate-400/50',
          iconBgClass: 'bg-slate-500/10',
          iconBorderClass: 'border-slate-500/20',
          iconColorClass: 'text-slate-600',
          textColorClass: 'text-slate-600 group-hover:text-slate-700',
          watermarkColorClass: 'text-slate-600/30',
        }
      );
    }

    return result;
  });

  constructor() {
    this.fetchAntrianSummary();
    this.klinikService.fetchKlinikInfo().subscribe();

    // Update live clock every 30 seconds
    const timer = setInterval(() => {
      this.now.set(new Date());
    }, 30000);

    this.destroyRef.onDestroy(() => {
      clearInterval(timer);
    });
  }

  private fetchAntrianSummary(): void {
    this.isLoadingAntrian.set(true);
    this.antrianService.getAntrian().subscribe({
      next: (items) => {
        this.antrianList.set(items || []);
        this.isLoadingAntrian.set(false);
      },
      error: () => {
        this.isLoadingAntrian.set(false);
      },
    });
  }
}
