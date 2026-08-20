import {
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  signal,
} from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucideChevronDown, lucideHistory, lucideUser } from '@ng-icons/lucide';
import { formatJakartaDate } from '../../../../../../core/utils/date.utils';
import { PriorityBadgeComponent } from '../../../../../../shared/components/priority-badge/priority-badge.component';
import { SensitiveValueComponent } from '../../../../../../shared/components/sensitive-value/sensitive-value.component';
import { HlmBadge } from '../../../../../../shared/ui/badge/src/lib/hlm-badge';
import { HlmButton } from '../../../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { KunjunganDetail } from '../../../../../antrian/antrian.types';
import { Pasien } from '../../../../../pasien/pasien.types';
import { RiwayatRekamMedisItem } from '../../../../rekam-medis.types';

@Component({
  selector: 'app-soap-patient-header',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    PriorityBadgeComponent,
    SensitiveValueComponent,
    HlmButton,
    HlmBadge,
    NgIcon,
    HlmIconDirective,
    ...HlmCardImports,
  ],
  providers: [
    provideIcons({
      lucideHistory,
      lucideChevronDown,
      lucideUser,
    }),
  ],
  templateUrl: './soap-patient-header.component.html',
})
export class SoapPatientHeaderComponent {
  readonly pasien = input<Pasien | null>(null);
  readonly kunjungan = input<KunjunganDetail | null>(null);
  readonly riwayatList = input<RiwayatRekamMedisItem[]>([]);

  readonly showHistory = signal<boolean>(false);

  readonly queueNumberDisplay = computed(() => {
    const num = this.kunjungan()?.nomorAntrian;
    return num != null ? String(num).padStart(3, '0') : '000';
  });

  toggleHistory(): void {
    this.showHistory.update((prev) => !prev);
  }

  formatDate(isoDateStr: string): string {
    return isoDateStr ? formatJakartaDate(isoDateStr) : '-';
  }
}
