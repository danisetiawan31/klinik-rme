import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideActivity,
  lucideAlertCircle,
  lucideCheckCircle2,
  lucideClock,
  lucideListOrdered,
  lucideUserCheck,
  lucideUserX,
  lucideUsers,
} from '@ng-icons/lucide';
import { HlmIconDirective } from '../../../../../shared/ui/icon/src/lib/hlm-icon.directive';

export type KpiColorVariant = 'primary' | 'emerald' | 'amber' | 'purple' | 'sky';

@Component({
  selector: 'app-landing-kpi-card',
  standalone: true,
  imports: [NgIcon, HlmIconDirective],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    provideIcons({
      lucideUsers,
      lucideUserCheck,
      lucideUserX,
      lucideCheckCircle2,
      lucideClock,
      lucideAlertCircle,
      lucideActivity,
      lucideListOrdered,
    }),
  ],
  templateUrl: './landing-kpi-card.component.html',
})
export class LandingKpiCardComponent {
  readonly label = input.required<string>();
  readonly value = input.required<number | string>();
  readonly unit = input<string>();
  readonly iconName = input.required<string>();
  readonly colorVariant = input<KpiColorVariant>('primary');

  readonly cardHoverBorderClass = computed(() => {
    switch (this.colorVariant()) {
      case 'emerald':
        return 'hover:border-emerald-500/40';
      case 'amber':
        return 'hover:border-amber-500/40';
      case 'purple':
        return 'hover:border-purple-500/40';
      case 'sky':
        return 'hover:border-sky-500/40';
      default:
        return 'hover:border-primary/40';
    }
  });

  readonly iconBoxClass = computed(() => {
    switch (this.colorVariant()) {
      case 'emerald':
        return 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20';
      case 'amber':
        return 'bg-amber-500/10 text-amber-600 border-amber-500/20';
      case 'purple':
        return 'bg-purple-500/10 text-purple-600 border-purple-500/20';
      case 'sky':
        return 'bg-sky-500/10 text-sky-600 border-sky-500/20';
      default:
        return 'bg-primary/10 text-primary border-primary/20';
    }
  });

  readonly labelColorClass = computed(() => {
    switch (this.colorVariant()) {
      case 'emerald':
        return 'text-emerald-700 dark:text-emerald-400 font-semibold';
      case 'amber':
        return 'text-amber-700 dark:text-amber-400 font-semibold';
      case 'purple':
        return 'text-purple-700 dark:text-purple-400 font-semibold';
      case 'sky':
        return 'text-sky-700 dark:text-sky-400 font-semibold';
      default:
        return 'text-muted-foreground';
    }
  });
}
