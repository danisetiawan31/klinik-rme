import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideCalendar,
  lucideClock,
  lucideSparkles,
  lucideStethoscope,
  lucideUserPlus,
  lucideShieldCheck,
} from '@ng-icons/lucide';
import { HlmIconDirective } from '../../../../../shared/ui/icon/src/lib/hlm-icon.directive';

@Component({
  selector: 'app-landing-hero',
  standalone: true,
  imports: [NgIcon, HlmIconDirective],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    provideIcons({
      lucideSparkles,
      lucideStethoscope,
      lucideUserPlus,
      lucideShieldCheck,
      lucideCalendar,
      lucideClock,
    }),
  ],
  templateUrl: './landing-hero.component.html',
})
export class LandingHeroComponent {
  readonly badgeIcon = input<string>('lucideSparkles');
  readonly badgeText = input.required<string>();
  readonly userName = input.required<string>();
  readonly subtitle = input.required<string>();
  readonly statusText = input.required<string>();
  readonly statusVariant = input<'open' | 'closed'>('open');
  readonly jamOperasional = input.required<string>();
  readonly dateStr = input.required<string>();
  readonly timeStr = input.required<string>();
}
