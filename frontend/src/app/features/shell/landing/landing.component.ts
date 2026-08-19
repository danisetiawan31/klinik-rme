import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  inject,
  signal,
} from '@angular/core';
import { ApexNonAxisChartSeries } from 'ng-apexcharts';
import { forkJoin, of, Subject } from 'rxjs';
import { debounceTime, distinctUntilChanged, switchMap } from 'rxjs/operators';
import { AuthService } from '../../../core/auth/auth.service';
import { KlinikService } from '../../../core/klinik/klinik.service';
import {
  formatJakartaDayDate,
  getJakartaISODate,
  getJakartaTimeString,
  getJakartaYesterdayISODate,
} from '../../../core/utils/date.utils';
import { AntrianService } from '../../antrian/antrian.service';
import { KunjunganListItem } from '../../antrian/antrian.types';
import { LaporanService } from '../../laporan/laporan.service';
import { LaporanHarian } from '../../laporan/laporan.types';
import { PasienService } from '../../pasien/pasien.service';
import { PasienSearchItem } from '../../pasien/pasien.types';

import { AdminDashboardViewComponent } from './components/admin-dashboard-view/admin-dashboard-view.component';
import { DoctorDashboardComponent } from './components/doctor-dashboard/doctor-dashboard.component';
import { PetugasDashboardComponent } from './components/petugas-dashboard/petugas-dashboard.component';

@Component({
  selector: 'app-landing',
  standalone: true,
  imports: [
    DoctorDashboardComponent,
    PetugasDashboardComponent,
    AdminDashboardViewComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './landing.component.html',
})
export class LandingComponent {
  private authService = inject(AuthService);
  private antrianService = inject(AntrianService);
  private klinikService = inject(KlinikService);
  private laporanService = inject(LaporanService);
  private pasienService = inject(PasienService);
  private destroyRef = inject(DestroyRef);

  readonly currentUser = this.authService.currentUser;
  readonly userName = computed(() => this.currentUser()?.nama || 'Pengguna');
  readonly userRoles = computed(() => this.currentUser()?.roles || []);

  readonly isDokter = computed(() => this.userRoles().includes('dokter'));
  readonly isPetugas = computed(() => this.userRoles().includes('petugas'));
  readonly isAdmin = computed(() => this.userRoles().includes('admin'));

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

  // Active called patient spotlight (for doctor immediate consultation)
  readonly activeCalledPatient = computed(() => {
    return this.antrianList().find((a) => a.status === 'dipanggil') || null;
  });

  // Filtered antrian for Doctor View
  readonly antrianMenungguList = computed(() => {
    return this.antrianList().filter((a) => a.status === 'menunggu');
  });
  readonly antrianSelesaiList = computed(() => {
    return this.antrianList().filter((a) => a.status === 'selesai');
  });

  // Doctor quick search state for patient EMR lookup
  readonly doctorSearchQuery = signal<string>('');
  readonly doctorSearchResults = signal<PasienSearchItem[]>([]);
  readonly isSearchingDoctor = signal<boolean>(false);
  private searchSubject = new Subject<string>();

  // Laporan Harian State
  readonly laporanHariIni = signal<LaporanHarian | null>(null);
  readonly laporanKemarin = signal<LaporanHarian | null>(null);
  readonly isLoadingLaporan = signal<boolean>(false);

  // Live formatted clock & date
  readonly currentDateStr = computed(() => formatJakartaDayDate(this.now()));
  readonly currentTimeStr = computed(() => `${getJakartaTimeString(this.now())} WIB`);

  // Computed summary metrics for Hero Banner
  readonly totalPasien = computed(() => this.antrianList().length);
  readonly antrianMenunggu = computed(
    () => this.antrianList().filter((a) => a.status === 'menunggu').length
  );
  readonly selesaiDilayani = computed(
    () => this.antrianList().filter((a) => a.status === 'selesai').length
  );
  readonly pasienPrioritas = computed(
    () => this.antrianList().filter((a) => a.isPriority).length
  );

  // Computed metrics for Performance Widget (Laporan Harian)
  readonly totalKunjunganLaporan = computed(() => {
    return this.laporanHariIni()?.totalKunjungan ?? this.totalPasien();
  });
  readonly totalSelesaiLaporan = computed(() => {
    return this.laporanHariIni()?.totalSelesai ?? this.selesaiDilayani();
  });
  readonly totalTidakHadirLaporan = computed(() => {
    return this.laporanHariIni()?.totalTidakHadir ?? 0;
  });

  readonly performanceRate = computed(() => {
    const total = this.totalKunjunganLaporan();
    if (!total) return 0;
    return Math.round((this.totalSelesaiLaporan() / total) * 100);
  });

  // Tingkat Kehadiran Pasien (Attendance Rate) dari Laporan Harian
  readonly attendanceRate = computed(() => {
    const total = this.totalKunjunganLaporan();
    if (!total) return 100;
    const tidakHadir = this.totalTidakHadirLaporan();
    return Math.max(0, Math.round(((total - tidakHadir) / total) * 100));
  });

  readonly trendVsKemarin = computed(() => {
    const hariIni = this.totalKunjunganLaporan();
    const kemarin = this.laporanKemarin()?.totalKunjungan ?? 0;
    if (kemarin === 0) {
      return { text: '– Data Awal', isPositive: null, icon: 'lucideMinus' as const };
    }
    const diff = hariIni - kemarin;
    if (diff === 0) {
      return { text: '0% vs kemarin', isPositive: null, icon: 'lucideMinus' as const };
    }
    const percent = Math.round((diff / kemarin) * 100);
    if (percent > 0) {
      return { text: `+${percent}% vs kemarin`, isPositive: true, icon: 'lucideTrendingUp' as const };
    }
    return { text: `${percent}% vs kemarin`, isPositive: false, icon: 'lucideTrendingDown' as const };
  });

  // ApexCharts Radial Gauge configuration
  readonly chartSeries = computed<ApexNonAxisChartSeries>(() => [this.performanceRate()]);
  readonly chartOptions = computed(() => ({
    chart: {
      type: 'radialBar' as const,
      height: 200,
      sparkline: { enabled: true },
      animations: {
        enabled: true,
        speed: 800,
        dynamicAnimation: { enabled: true, speed: 350 },
      },
    },
    plotOptions: {
      radialBar: {
        startAngle: -135,
        endAngle: 135,
        hollow: {
          margin: 0,
          size: '72%',
          background: 'transparent',
        },
        track: {
          background: 'color-mix(in srgb, var(--primary) 12%, transparent)',
          strokeWidth: '97%',
          margin: 0,
        },
        dataLabels: {
          name: {
            show: true,
            fontSize: '11px',
            fontWeight: 600,
            color: 'var(--color-muted-foreground)',
            offsetY: -8,
          },
          value: {
            offsetY: 6,
            fontSize: '22px',
            fontWeight: 700,
            color: 'var(--color-foreground)',
            formatter: (val: number) => `${val}%`,
          },
        },
      },
    },
    fill: {
      type: 'gradient' as const,
      gradient: {
        shade: 'dark' as const,
        type: 'horizontal' as const,
        shadeIntensity: 0.5,
        gradientToColors: ['var(--accent)'],
        inverseColors: true,
        opacityFrom: 1,
        opacityTo: 1,
        stops: [0, 100],
      },
    },
    stroke: {
      lineCap: 'round' as const,
    },
    colors: ['var(--primary)'],
    labels: ['Tingkat Layanan'],
  }));

  constructor() {
    this.fetchAntrianSummary();
    this.fetchLaporanSummary();
    this.klinikService.fetchKlinikInfo().subscribe();

    // Doctor quick search reactive pipeline
    const searchSub = this.searchSubject
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        switchMap((query) => {
          const trimmed = query.trim();
          if (!trimmed) {
            this.isSearchingDoctor.set(false);
            this.doctorSearchResults.set([]);
            return of({ items: [], totalCount: 0 });
          }
          this.isSearchingDoctor.set(true);
          const isNum = /^\d+$/.test(trimmed);
          const params = isNum
            ? { nik: trimmed, page: 1, limit: 4 }
            : { nama: trimmed, page: 1, limit: 4 };
          return this.pasienService.search(params);
        })
      )
      .subscribe({
        next: (res) => {
          this.doctorSearchResults.set(res.items || []);
          this.isSearchingDoctor.set(false);
        },
        error: () => {
          this.isSearchingDoctor.set(false);
        },
      });

    // Update live clock every 30 seconds
    const timer = setInterval(() => {
      this.now.set(new Date());
    }, 30000);

    this.destroyRef.onDestroy(() => {
      clearInterval(timer);
      searchSub.unsubscribe();
    });
  }

  onDoctorSearchInput(val: string): void {
    this.doctorSearchQuery.set(val);
    this.searchSubject.next(val);
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

  private fetchLaporanSummary(): void {
    this.isLoadingLaporan.set(true);
    const today = getJakartaISODate(this.now());
    const yesterday = getJakartaYesterdayISODate(this.now());

    forkJoin({
      hariIni: this.laporanService.getLaporanHarian(today),
      kemarin: this.laporanService.getLaporanHarian(yesterday),
    }).subscribe({
      next: ({ hariIni, kemarin }) => {
        this.laporanHariIni.set(hariIni);
        this.laporanKemarin.set(kemarin);
        this.isLoadingLaporan.set(false);
      },
      error: () => {
        this.isLoadingLaporan.set(false);
      },
    });
  }
}
