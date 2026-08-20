import {
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
} from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideAccessibility,
  lucideActivity,
  lucideBaby,
  lucideHeart,
  lucideStar,
  lucideUserCheck,
} from '@ng-icons/lucide';
import { HlmBadge } from '../../ui/badge/src/lib/hlm-badge';
import { HlmIconDirective } from '../../ui/icon/src/lib/hlm-icon.directive';

export type PriorityBadgeSize = 'default' | 'sm' | 'xs';

@Component({
  selector: 'app-priority-badge',
  standalone: true,
  imports: [HlmBadge, NgIcon, HlmIconDirective],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    provideIcons({
      lucideStar,
      lucideUserCheck,
      lucideBaby,
      lucideHeart,
      lucideAccessibility,
      lucideActivity,
    }),
  ],
  templateUrl: './priority-badge.component.html',
})
export class PriorityBadgeComponent {
  reason = input<string | null | undefined>();
  size = input<PriorityBadgeSize>('default');
  showReason = input<boolean>(false);

  readonly iconName = computed(() => {
    const r = (this.reason() || '').toLowerCase();
    if (r.includes('lansia')) return 'lucideUserCheck';
    if (r.includes('bayi') || r.includes('balita') || r.includes('anak')) return 'lucideBaby';
    if (r.includes('hamil')) return 'lucideHeart';
    if (r.includes('disabilitas') || r.includes('kursi roda')) return 'lucideAccessibility';
    if (r.includes('darurat') || r.includes('kritis') || r.includes('medis')) return 'lucideActivity';
    return 'lucideStar';
  });

  readonly displayLabel = computed(() => {
    if (this.showReason() && this.reason()) {
      return this.reason()!;
    }
    return 'Prioritas';
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
