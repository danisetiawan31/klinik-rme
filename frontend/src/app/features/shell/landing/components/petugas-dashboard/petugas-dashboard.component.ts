import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideAlertCircle,
  lucideCheckCircle2,
  lucideClock,
  lucideInbox,
  lucideListOrdered,
  lucideUser,
  lucideUserPlus,
  lucideUsers,
} from '@ng-icons/lucide';
import { KunjunganListItem } from '../../../../antrian/antrian.types';
import { PriorityBadgeComponent } from '../../../../../shared/components/priority-badge/priority-badge.component';
import { StatusBadgeComponent } from '../../../../../shared/components/status-badge/status-badge.component';
import { HlmEmptyImports } from '../../../../../shared/ui/empty/src/lib/hlm-empty';
import { HlmIconDirective } from '../../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { LandingHeroComponent } from '../landing-hero/landing-hero.component';
import { LandingKpiCardComponent } from '../landing-kpi-card/landing-kpi-card.component';

@Component({
  selector: 'app-petugas-dashboard',
  standalone: true,
  imports: [
    RouterLink,
    NgIcon,
    HlmIconDirective,
    HlmEmptyImports,
    StatusBadgeComponent,
    PriorityBadgeComponent,
    LandingHeroComponent,
    LandingKpiCardComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    provideIcons({
      lucideUserPlus,
      lucideListOrdered,
      lucideUsers,
      lucideClock,
      lucideCheckCircle2,
      lucideAlertCircle,
      lucideInbox,
      lucideUser,
    }),
  ],
  templateUrl: './petugas-dashboard.component.html',
})
export class PetugasDashboardComponent {
  readonly userName = input.required<string>();
  readonly isKlinikBuka = input.required<boolean>();
  readonly jamOperasionalStr = input.required<string>();
  readonly currentDateStr = input.required<string>();
  readonly currentTimeStr = input.required<string>();
  readonly totalPasien = input.required<number>();
  readonly antrianMenunggu = input.required<number>();
  readonly selesaiDilayani = input.required<number>();
  readonly pasienPrioritas = input.required<number>();
  readonly antrianList = input.required<KunjunganListItem[]>();

  formatQueue(num: number | undefined | null): string {
    return num != null ? String(num).padStart(3, '0') : '000';
  }
}
