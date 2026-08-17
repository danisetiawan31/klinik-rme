import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  OnInit,
  signal,
  viewChild,
} from '@angular/core';
import { toast } from '@spartan-ng/brain/sonner';
import { AuthService } from '../../core/auth/auth.service';
import { KlinikService } from '../../core/klinik/klinik.service';
import { RealtimeService } from '../../core/realtime/realtime.service';
import { ClinicStatusIndicatorComponent } from '../../shared/components/clinic-status-indicator/clinic-status-indicator.component';
import { ConnectionStatusIndicatorComponent } from '../../shared/components/connection-status-indicator/connection-status-indicator.component';
import { PriorityBadgeComponent } from '../../shared/components/priority-badge/priority-badge.component';
import { StatusBadgeComponent } from '../../shared/components/status-badge/status-badge.component';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucideAlertTriangle, lucideInbox, lucideMegaphone } from '@ng-icons/lucide';
import { HlmButton } from '../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../shared/ui/card/src/index';
import { HlmDialog } from '../../shared/ui/dialog/src/lib/hlm-dialog';
import { HlmDialogImports } from '../../shared/ui/dialog/src/index';
import { HlmEmptyImports } from '../../shared/ui/empty/src/index';
import { HlmIconDirective } from '../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmSkeletonImports } from '../../shared/ui/skeleton/src/index';
import { HlmTableImports } from '../../shared/ui/table/src/index';
import { AntrianService } from './antrian.service';
import { KunjunganListItem } from './antrian.types';

@Component({
  selector: 'app-antrian-dashboard',
  standalone: true,
  imports: [
    StatusBadgeComponent,
    PriorityBadgeComponent,
    ClinicStatusIndicatorComponent,
    ConnectionStatusIndicatorComponent,
    HlmButton,
    NgIcon,
    HlmIconDirective,
    ...HlmCardImports,
    ...HlmDialogImports,
    ...HlmEmptyImports,
    ...HlmSkeletonImports,
    ...HlmTableImports,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [provideIcons({ lucideMegaphone, lucideInbox, lucideAlertTriangle })],
  templateUrl: './antrian-dashboard.component.html',
})
export class AntrianDashboardComponent implements OnInit {
  private antrianService = inject(AntrianService);
  private realtimeService = inject(RealtimeService);
  private klinikService = inject(KlinikService);
  private authService = inject(AuthService);
  private destroyRef = inject(DestroyRef);

  // ViewChild reference to Spartan Dialog for programmatic open/close
  readonly confirmDialog = viewChild<HlmDialog>('confirmTidakHadirDialog');

  readonly antrianList = signal<KunjunganListItem[]>([]);
  readonly isLoading = signal<boolean>(false);
  readonly isSubmittingAction = signal<boolean>(false);

  // Toast feedback state for testing & programmatic feedback
  readonly toastMessage = signal<string | null>(null);
  readonly toastType = signal<'error' | 'success' | 'info'>('info');

  // Confirmation dialog state for "Tandai Tidak Hadir" (final action)
  readonly confirmTidakHadirKunjungan = signal<KunjunganListItem | null>(null);

  // RBAC computed signals
  readonly currentUser = this.authService.currentUser;
  readonly isDokter = computed(() => this.currentUser()?.roles.includes('dokter') ?? false);
  readonly isAdmin = computed(() => this.currentUser()?.roles.includes('admin') ?? false);
  readonly isPetugas = computed(() => this.currentUser()?.roles.includes('petugas') ?? false);
  readonly hasAnyAction = computed(() => this.isDokter() || this.isAdmin());

  // Live WebSocket state & clinic info
  readonly connectionStatus = this.realtimeService.connectionStatus;
  readonly lastUpdateAt = this.realtimeService.lastUpdateAt;
  readonly klinikInfo = this.klinikService.klinikInfo;
  readonly isBuka = computed(() => {
    this.lastUpdateAt();
    return this.klinikService.isKlinikBuka(this.klinikInfo());
  });

  // Client-side sorting: isPriority DESC, skipCount ASC, nomorAntrian ASC
  readonly sortedAntrian = computed(() => {
    return [...this.antrianList()].sort((a, b) => {
      const priorityDiff = (b.isPriority ? 1 : 0) - (a.isPriority ? 1 : 0);
      if (priorityDiff !== 0) return priorityDiff;

      const skipDiff = (a.skipCount ?? 0) - (b.skipCount ?? 0);
      if (skipDiff !== 0) return skipDiff;

      return a.nomorAntrian - b.nomorAntrian;
    });
  });

  // Summary statistics computed signals
  readonly totalCount = computed(() => this.antrianList().length);
  readonly menungguCount = computed(
    () => this.antrianList().filter((k) => k.status === 'menunggu').length
  );
  readonly dipanggilCount = computed(
    () => this.antrianList().filter((k) => k.status === 'dipanggil').length
  );
  readonly selesaiCount = computed(
    () => this.antrianList().filter((k) => k.status === 'selesai').length
  );

  constructor() {
    this.destroyRef.onDestroy(() => {
      this.realtimeService.disconnect();
    });

    effect(() => {
      const updateTime = this.lastUpdateAt();
      const status = this.connectionStatus();

      if (updateTime !== null || status === 'connected') {
        this.loadAntrian();
      }
    });
  }

  ngOnInit(): void {
    this.realtimeService.connect();

    if (!this.klinikInfo()) {
      this.klinikService.fetchKlinikInfo().subscribe();
    }

    this.loadAntrian();
  }

  loadAntrian(): void {
    this.isLoading.set(true);
    this.antrianService.getAntrian().subscribe({
      next: (items) => {
        this.antrianList.set(items);
        this.isLoading.set(false);
      },
      error: (err) => {
        let msg = 'Gagal memuat daftar antrian';
        if (err?.error?.message) {
          msg = err.error.message;
        } else if (err?.error?.error?.message) {
          msg = err.error.error.message;
        }
        this.toastType.set('error');
        this.toastMessage.set(msg);
        toast.error(msg);
        this.isLoading.set(false);
      },
    });
  }

  /**
   * Action 1: Dokter memanggil pasien berikutnya (POST /klinik/:id/panggil-berikutnya)
   */
  onPanggilBerikutnya(): void {
    if (this.isSubmittingAction()) return;
    this.isSubmittingAction.set(true);

    this.antrianService.panggilBerikutnya().subscribe({
      next: (res) => {
        this.isSubmittingAction.set(false);
        if (!res) {
          this.toastType.set('info');
          this.toastMessage.set('Antrian kosong, tidak ada pasien menunggu');
          toast.info('Antrian kosong, tidak ada pasien menunggu');
        } else {
          const msg = `Berhasil memanggil antrian #${res.nomorAntrian} (${res.pasienNama})`;
          this.toastType.set('success');
          this.toastMessage.set(msg);
          toast.success(msg);
          this.loadAntrian();
        }
      },
      error: (err) => {
        this.isSubmittingAction.set(false);
        let msg = 'Gagal memanggil pasien berikutnya';
        if (err?.error?.message) {
          msg = err.error.message;
        } else if (err?.error?.error?.message) {
          msg = err.error.error.message;
        }
        this.toastType.set('error');
        this.toastMessage.set(msg);
        toast.error(msg);
      },
    });
  }

  /**
   * Action 2: Dokter melewati pasien yang tidak datang saat dipanggil (POST /kunjungan/:id/lewati)
   */
  onLewati(item: KunjunganListItem): void {
    if (this.isSubmittingAction()) return;
    this.isSubmittingAction.set(true);

    this.antrianService.lewati(item.id).subscribe({
      next: () => {
        this.isSubmittingAction.set(false);
        const msg = `Pasien antrian #${item.nomorAntrian} dilewati dan dikembalikan ke antrian`;
        this.toastType.set('success');
        this.toastMessage.set(msg);
        toast.success(msg);
        this.loadAntrian();
      },
      error: (err) => {
        this.isSubmittingAction.set(false);
        let msg = 'Gagal melewati antrian pasien';
        if (err?.error?.message) {
          msg = err.error.message;
        } else if (err?.error?.error?.message) {
          msg = err.error.error.message;
        }
        this.toastType.set('error');
        this.toastMessage.set(msg);
        toast.error(msg);
        this.loadAntrian();
      },
    });
  }

  /**
   * Action 3: Buka dialog konfirmasi untuk menandai tidak hadir
   */
  openConfirmTidakHadir(item: KunjunganListItem): void {
    this.confirmTidakHadirKunjungan.set(item);
    this.confirmDialog()?.open();
  }

  cancelConfirmTidakHadir(): void {
    this.confirmTidakHadirKunjungan.set(null);
    this.confirmDialog()?.close();
  }

  /**
   * Action 3 Execution: Dokter/Admin submit tandai tidak hadir (POST /kunjungan/:id/tidak-hadir)
   */
  executeTidakHadir(): void {
    const target = this.confirmTidakHadirKunjungan();
    if (!target || this.isSubmittingAction()) return;

    this.isSubmittingAction.set(true);
    this.antrianService.tidakHadir(target.id).subscribe({
      next: () => {
        this.isSubmittingAction.set(false);
        this.confirmTidakHadirKunjungan.set(null);
        this.confirmDialog()?.close();
        const msg = `Pasien antrian #${target.nomorAntrian} (${target.pasienNama}) ditandai tidak hadir`;
        this.toastType.set('success');
        this.toastMessage.set(msg);
        toast.success(msg);
        this.loadAntrian();
      },
      error: (err) => {
        this.isSubmittingAction.set(false);
        this.confirmTidakHadirKunjungan.set(null);
        this.confirmDialog()?.close();
        let msg = 'Gagal menandai pasien tidak hadir';
        if (err?.error?.message) {
          msg = err.error.message;
        } else if (err?.error?.error?.message) {
          msg = err.error.error.message;
        }
        this.toastType.set('error');
        this.toastMessage.set(msg);
        toast.error(msg);
        this.loadAntrian();
      },
    });
  }
}
