import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { HlmBadge } from '@spartan-ng/helm/badge';

@Component({
  selector: 'app-priority-badge',
  standalone: true,
  imports: [HlmBadge],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './priority-badge.component.html',
})
export class PriorityBadgeComponent {
  reason = input<string | null | undefined>();
}
