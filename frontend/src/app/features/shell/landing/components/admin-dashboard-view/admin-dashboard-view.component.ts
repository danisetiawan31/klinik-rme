import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideActivity,
  lucideCalendarDays,
  lucideInbox,
  lucideKey,
  lucideMinus,
  lucideShieldCheck,
  lucideTrendingDown,
  lucideTrendingUp,
  lucideUserCheck,
  lucideUsers,
} from '@ng-icons/lucide';
import { ApexNonAxisChartSeries, NgApexchartsModule } from 'ng-apexcharts';
import { KunjunganListItem } from '../../../../antrian/antrian.types';
import { StatusBadgeComponent } from '../../../../../shared/components/status-badge/status-badge.component';
import { HlmEmptyImports } from '../../../../../shared/ui/empty/src/lib/hlm-empty';
import { HlmIconDirective } from '../../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { LandingHeroComponent } from '../landing-hero/landing-hero.component';
import { LandingKpiCardComponent } from '../landing-kpi-card/landing-kpi-card.component';

@Component({
  selector: 'app-admin-dashboard-view',
  standalone: true,
  imports: [
    RouterLink,
    NgIcon,
    HlmIconDirective,
    HlmEmptyImports,
    StatusBadgeComponent,
    NgApexchartsModule,
    LandingHeroComponent,
    LandingKpiCardComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    provideIcons({
      lucideShieldCheck,
      lucideKey,
      lucideUserCheck,
      lucideUsers,
      lucideActivity,
      lucideCalendarDays,
      lucideInbox,
      lucideTrendingUp,
      lucideTrendingDown,
      lucideMinus,
    }),
  ],
  templateUrl: './admin-dashboard-view.component.html',
})
export class AdminDashboardViewComponent {
  readonly userName = input.required<string>();
  readonly isKlinikBuka = input.required<boolean>();
  readonly jamOperasionalStr = input.required<string>();
  readonly currentDateStr = input.required<string>();
  readonly currentTimeStr = input.required<string>();
  readonly totalPasien = input.required<number>();
  readonly attendanceRate = input.required<number>();
  readonly performanceRate = input.required<number>();
  readonly totalTidakHadirLaporan = input.required<number>();
  readonly antrianList = input.required<KunjunganListItem[]>();
  readonly chartSeries = input.required<ApexNonAxisChartSeries>();
  readonly chartOptions = input.required<any>();
  readonly trendVsKemarin = input.required<{ text: string; isPositive: boolean | null; icon: string }>();
}
