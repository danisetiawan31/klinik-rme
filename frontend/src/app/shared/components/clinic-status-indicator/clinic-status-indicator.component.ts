import { ChangeDetectionStrategy, Component, computed, inject, OnInit } from '@angular/core';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { KlinikService } from '../../../core/klinik/klinik.service';

@Component({
  selector: 'app-clinic-status-indicator',
  standalone: true,
  imports: [HlmBadge],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './clinic-status-indicator.component.html',
})
export class ClinicStatusIndicatorComponent implements OnInit {
  private klinikService = inject(KlinikService);

  readonly klinikInfo = this.klinikService.klinikInfo;
  readonly isBuka = computed(() => this.klinikService.isKlinikBuka(this.klinikInfo()));

  ngOnInit(): void {
    if (!this.klinikInfo()) {
      this.klinikService.fetchKlinikInfo().subscribe();
    }
  }
}
