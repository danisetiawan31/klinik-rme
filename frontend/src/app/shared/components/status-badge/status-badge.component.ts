import {
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
} from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideCheckCircle2,
  lucideClock,
  lucideMegaphone,
  lucideUserX,
} from '@ng-icons/lucide';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { KunjunganStatus } from '../../../features/antrian/antrian.types';
import { HlmIconDirective } from '../../ui/icon/src/lib/hlm-icon.directive';

export type StatusBadgeSize = 'default' | 'sm' | 'xs';

@Component({
  selector: 'app-status-badge',
  standalone: true,
  imports: [HlmBadge, NgIcon, HlmIconDirective],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    provideIcons({
      lucideClock,
      lucideMegaphone,
      lucideCheckCircle2,
      lucideUserX,
    }),
  ],
  templateUrl: './status-badge.component.html',
})
export class StatusBadgeComponent {
  status = input<KunjunganStatus>('menunggu');
  size = input<StatusBadgeSize>('default');

  readonly badgeConfig = computed(() => {
    switch (this.status()) {
      case 'menunggu':
        return {
          label: 'Menunggu',
          iconName: 'lucideClock',
          class:
            'bg-amber-500/10 border border-amber-500/30 text-amber-700 dark:text-amber-300',
          iconClass: 'text-amber-600 dark:text-amber-400',
        };
      case 'dipanggil':
        return {
          label: 'Dipanggil',
          iconName: 'lucideMegaphone',
          class:
            'bg-primary border border-primary/50 text-primary-foreground shadow-[0_0_10px_color-mix(in_srgb,var(--primary)_35%,transparent)] animate-pulse',
          iconClass: 'text-primary-foreground',
        };
      case 'selesai':
        return {
          label: 'Selesai',
          iconName: 'lucideCheckCircle2',
          class:
            'bg-emerald-500/10 border border-emerald-500/30 text-emerald-700 dark:text-emerald-300',
          iconClass: 'text-emerald-600 dark:text-emerald-400',
        };
      case 'tidak_hadir':
        return {
          label: 'Tidak Hadir',
          iconName: 'lucideUserX',
          class:
            'bg-muted/80 border border-border text-muted-foreground',
          iconClass: 'text-muted-foreground',
        };
      default:
        return {
          label: this.status() as string,
          iconName: 'lucideClock',
          class:
            'bg-muted border border-border text-muted-foreground',
          iconClass: 'text-muted-foreground',
        };
    }
  });

  readonly sizeClass = computed(() => {
    switch (this.size()) {
      case 'xs':
        return 'size-6 p-0 rounded-md inline-flex items-center justify-center';
      case 'sm':
        return 'px-2 py-0.5 text-[11px] gap-1 rounded-md inline-flex items-center font-medium';
      case 'default':
      default:
        return 'px-2.5 py-1 text-xs gap-1.5 rounded-md inline-flex items-center font-medium';
    }
  });
}
