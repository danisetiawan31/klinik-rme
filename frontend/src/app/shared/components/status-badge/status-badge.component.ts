import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { KunjunganStatus } from '../../../features/antrian/antrian.types';

@Component({
  selector: 'app-status-badge',
  standalone: true,
  imports: [HlmBadge],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './status-badge.component.html',
})
export class StatusBadgeComponent {
  status = input<KunjunganStatus>('menunggu');

  readonly badgeConfig = computed(() => {
    switch (this.status()) {
      case 'menunggu':
        return {
          label: 'Menunggu',
          class: 'bg-muted border border-warning text-warning-foreground',
          dotClass: 'bg-warning',
        };
      case 'dipanggil':
        return {
          label: 'Dipanggil',
          class: 'bg-primary border-transparent text-primary-foreground animate-pulse',
          dotClass: 'bg-primary-foreground',
        };
      case 'selesai':
        return {
          label: 'Selesai',
          class: 'bg-accent border-transparent text-accent-foreground',
          dotClass: 'bg-accent-foreground',
        };
      case 'tidak_hadir':
        return {
          label: 'Tidak Hadir',
          class: 'bg-muted border border-border text-muted-foreground',
          dotClass: 'bg-muted-foreground',
        };
      default:
        return {
          label: this.status() as string,
          class: 'bg-muted border border-border text-muted-foreground',
          dotClass: 'bg-muted-foreground',
        };
    }
  });
}
