import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucideFlag } from '@ng-icons/lucide';
import { HlmBadge } from '../../ui/badge/src/lib/hlm-badge';
import { HlmIconDirective } from '../../ui/icon/src/lib/hlm-icon.directive';

@Component({
  selector: 'app-priority-badge',
  standalone: true,
  imports: [HlmBadge, NgIcon, HlmIconDirective],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [provideIcons({ lucideFlag })],
  templateUrl: './priority-badge.component.html',
})
export class PriorityBadgeComponent {
  reason = input<string | null | undefined>();
}
