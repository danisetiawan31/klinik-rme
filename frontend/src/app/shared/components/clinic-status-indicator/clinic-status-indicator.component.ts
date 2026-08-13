import { ChangeDetectionStrategy, Component, computed, inject, OnInit } from '@angular/core';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { KlinikService } from '../../../core/klinik/klinik.service';

@Component({
  selector: 'app-clinic-status-indicator',
  standalone: true,
  imports: [HlmBadge],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (isBuka()) {
      <span
        hlmBadge
        variant="outline"
        class="bg-[#F0FDF4] border-[#BBF7D0] text-[var(--color-accent)] font-medium px-2.5 py-1 flex items-center gap-1.5"
        aria-label="Status Klinik: Buka"
      >
        <span class="h-2 w-2 rounded-full bg-[var(--color-accent)]"></span>
        Klinik Buka
      </span>
    } @else {
      <span
        hlmBadge
        variant="outline"
        class="bg-[var(--color-muted)] border-[var(--color-border)] text-[var(--color-muted-foreground)] font-medium px-2.5 py-1 flex items-center gap-1.5"
        aria-label="Status Klinik: Tutup"
      >
        <span class="h-2 w-2 rounded-full bg-[var(--color-muted-foreground)]"></span>
        Klinik Tutup
      </span>
    }
  `,
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
