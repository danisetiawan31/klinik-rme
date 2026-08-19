import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideAlertTriangle,
  lucideBuilding,
  lucideCheck,
  lucideClock,
  lucideHelpCircle,
  lucideKey,
  lucideRefreshCw,
  lucideShieldAlert,
  lucideTv,
} from '@ng-icons/lucide';
import { toast } from '@spartan-ng/brain/sonner';
import { KlinikService } from '../../../../core/klinik/klinik.service';
import { ClinicStatusIndicatorComponent } from '../../../../shared/components/clinic-status-indicator/clinic-status-indicator.component';
import { RevealOnceSecretComponent } from '../../../../shared/components/reveal-once-secret/reveal-once-secret.component';
import { HlmAlertImports } from '../../../../shared/ui/alert/src/index';
import { HlmBadge } from '../../../../shared/ui/badge/src/lib/hlm-badge';
import { HlmButton } from '../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmSkeletonImports } from '../../../../shared/ui/skeleton/src/index';
import { AdminService } from '../../admin.service';

@Component({
  selector: 'app-admin-klinik',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    RevealOnceSecretComponent,
    ClinicStatusIndicatorComponent,
    HlmButton,
    HlmBadge,
    NgIcon,
    HlmIconDirective,
    ...HlmAlertImports,
    ...HlmCardImports,
    ...HlmSkeletonImports,
  ],
  providers: [
    provideIcons({
      lucideBuilding,
      lucideKey,
      lucideClock,
      lucideTv,
      lucideRefreshCw,
      lucideShieldAlert,
      lucideAlertTriangle,
      lucideCheck,
      lucideHelpCircle,
    }),
  ],
  templateUrl: './admin-klinik.component.html',
})
export class AdminKlinikComponent implements OnInit {
  private adminService = inject(AdminService);
  readonly klinikService = inject(KlinikService);

  readonly showConfirmModal = signal<boolean>(false);
  readonly isRegenerating = signal<boolean>(false);
  readonly newDisplayToken = signal<string | null>(null);

  ngOnInit(): void {
    this.klinikService.fetchKlinikInfo().subscribe();
  }

  openConfirmModal(): void {
    this.showConfirmModal.set(true);
  }

  closeConfirmModal(): void {
    this.showConfirmModal.set(false);
  }

  confirmRegenerate(): void {
    const klinikId = this.klinikService.klinikInfo()?.id || 1;
    this.isRegenerating.set(true);

    this.adminService.regenerateDisplayToken(klinikId).subscribe({
      next: (res) => {
        this.isRegenerating.set(false);
        this.closeConfirmModal();
        this.newDisplayToken.set(res.displayToken);
        toast.success('Display token berhasil dibuat ulang!');
      },
      error: (err) => {
        this.isRegenerating.set(false);
        toast.error(err?.error?.message || 'Gagal membuat ulang display token.');
      },
    });
  }
}
