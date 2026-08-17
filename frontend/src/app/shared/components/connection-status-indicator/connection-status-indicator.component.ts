import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
} from '@angular/core';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { ConnectionStatus, RealtimeService } from '../../../core/realtime/realtime.service';

@Component({
  selector: 'app-connection-status-indicator',
  standalone: true,
  imports: [HlmBadge],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './connection-status-indicator.component.html',
})
export class ConnectionStatusIndicatorComponent {
  private realtimeService = inject(RealtimeService);

  /**
   * Optional manual override for status input.
   * If not provided, it falls back to the reactive signal from RealtimeService.
   */
  readonly statusOverride = input<ConnectionStatus | undefined>(undefined, {
    alias: 'status',
  });

  readonly currentStatus = computed<ConnectionStatus>(() => {
    return this.statusOverride() ?? this.realtimeService.connectionStatus();
  });
}
