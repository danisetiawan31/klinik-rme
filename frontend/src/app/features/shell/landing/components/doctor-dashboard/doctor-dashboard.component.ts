import { ChangeDetectionStrategy, Component, input, output, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideCheckCircle2,
  lucideFileText,
  lucideInbox,
  lucideListOrdered,
  lucideSearch,
  lucideStethoscope,
  lucideTrendingUp,
} from '@ng-icons/lucide';
import { KunjunganListItem } from '../../../../antrian/antrian.types';
import { PasienSearchItem } from '../../../../pasien/pasien.types';
import { PriorityBadgeComponent } from '../../../../../shared/components/priority-badge/priority-badge.component';
import { HlmEmptyImports } from '../../../../../shared/ui/empty/src/lib/hlm-empty';
import { HlmIconDirective } from '../../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { LandingHeroComponent } from '../landing-hero/landing-hero.component';
import { LandingKpiCardComponent } from '../landing-kpi-card/landing-kpi-card.component';

@Component({
  selector: 'app-doctor-dashboard',
  standalone: true,
  imports: [
    RouterLink,
    NgIcon,
    HlmIconDirective,
    HlmEmptyImports,
    PriorityBadgeComponent,
    LandingHeroComponent,
    LandingKpiCardComponent,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    provideIcons({
      lucideStethoscope,
      lucideCheckCircle2,
      lucideListOrdered,
      lucideInbox,
      lucideSearch,
      lucideFileText,
      lucideTrendingUp,
    }),
  ],
  templateUrl: './doctor-dashboard.component.html',
})
export class DoctorDashboardComponent {
  readonly userName = input.required<string>();
  readonly isKlinikBuka = input.required<boolean>();
  readonly jamOperasionalStr = input.required<string>();
  readonly currentDateStr = input.required<string>();
  readonly currentTimeStr = input.required<string>();
  readonly selesaiDilayani = input.required<number>();
  readonly antrianMenunggu = input.required<number>();
  readonly pasienPrioritas = input.required<number>();
  readonly totalPasien = input.required<number>();

  readonly activeCalledPatient = input<KunjunganListItem | null>(null);
  readonly antrianMenungguList = input.required<KunjunganListItem[]>();
  readonly antrianSelesaiList = input.required<KunjunganListItem[]>();

  readonly doctorSearchQuery = input.required<string>();
  readonly doctorSearchResults = input.required<PasienSearchItem[]>();
  readonly isSearchingDoctor = input.required<boolean>();

  readonly searchInput = output<string>();

  readonly activeTab = signal<'menunggu' | 'selesai'>('menunggu');

  setTab(tab: 'menunggu' | 'selesai'): void {
    this.activeTab.set(tab);
  }

  onSearch(val: string): void {
    this.searchInput.emit(val);
  }

  formatQueue(num: number | undefined | null): string {
    return num != null ? String(num).padStart(3, '0') : '000';
  }
}
