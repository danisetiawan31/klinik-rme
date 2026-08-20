import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideActivity,
  lucideAlertCircle,
  lucideCheckCircle2,
  lucideClock,
  lucideKey,
  lucideMegaphone,
  lucideSparkles,
  lucideUsers,
} from '@ng-icons/lucide';
import { environment } from '../../../environments/environment';
import { KlinikService } from '../../core/klinik/klinik.service';
import { RealtimeService } from '../../core/realtime/realtime.service';
import {
  formatJakartaDayDate,
  getJakartaTimeString,
} from '../../core/utils/date.utils';
import { ConnectionStatusIndicatorComponent } from '../../shared/components/connection-status-indicator/connection-status-indicator.component';
import { PriorityBadgeComponent } from '../../shared/components/priority-badge/priority-badge.component';
import { StatusBadgeComponent } from '../../shared/components/status-badge/status-badge.component';
import { HlmAlertImports } from '../../shared/ui/alert/src/index';
import { HlmBadge } from '../../shared/ui/badge/src/lib/hlm-badge';
import { HlmButton } from '../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../shared/ui/card/src/index';
import { HlmDialogImports } from '../../shared/ui/dialog/src/index';
import { HlmIconDirective } from '../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../shared/ui/input/src/lib/hlm-input';
import { HlmLabel } from '../../shared/ui/label/src/lib/hlm-label';
import { HlmSkeletonImports } from '../../shared/ui/skeleton/src/index';
import { AntrianService } from '../antrian/antrian.service';
import { KunjunganListItem } from '../antrian/antrian.types';

export const DISPLAY_TOKEN_STORAGE_KEY = 'klinik_display_token';

@Component({
  selector: 'app-papan-antrian',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FormsModule,
    ConnectionStatusIndicatorComponent,
    PriorityBadgeComponent,
    StatusBadgeComponent,
    HlmButton,
    HlmInput,
    HlmLabel,
    HlmBadge,
    NgIcon,
    HlmIconDirective,
    ...HlmAlertImports,
    ...HlmCardImports,
    ...HlmDialogImports,
    ...HlmSkeletonImports,
  ],
  providers: [
    provideIcons({
      lucideMegaphone,
      lucideClock,
      lucideActivity,
      lucideKey,
      lucideCheckCircle2,
      lucideUsers,
      lucideAlertCircle,
      lucideSparkles,
    }),
  ],
  templateUrl: './papan-antrian.component.html',
})
export class PapanAntrianComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private antrianService = inject(AntrianService);
  private realtimeService = inject(RealtimeService);
  private klinikService = inject(KlinikService);
  private destroyRef = inject(DestroyRef);

  readonly klinikInfo = this.klinikService.klinikInfo;
  readonly antrianList = signal<KunjunganListItem[]>([]);
  readonly isLoading = signal<boolean>(true);
  readonly errorMessage = signal<string | null>(null);

  // Token management
  readonly displayToken = signal<string>('');
  readonly showTokenModal = signal<boolean>(false);
  readonly tokenInput = signal<string>('');

  // Live Clock
  readonly currentTimeString = signal<string>('');
  readonly currentDayDateString = signal<string>('');

  // Realtime signals
  readonly connectionStatus = this.realtimeService.connectionStatus;
  readonly lastUpdateAt = this.realtimeService.lastUpdateAt;

  // Active called patient (Status: 'dipanggil')
  readonly activeCalling = computed<KunjunganListItem | null>(() => {
    const list = this.antrianList();
    return list.find((item) => item.status === 'dipanggil') || null;
  });

  // Waiting list patients (Status: 'menunggu')
  readonly waitingList = computed<KunjunganListItem[]>(() => {
    const list = this.antrianList();
    return list.filter((item) => item.status === 'menunggu');
  });

  // Completed & stats
  readonly totalSelesai = computed<number>(() => {
    return this.antrianList().filter((item) => item.status === 'selesai').length;
  });

  constructor() {
    // Dual-trigger reactivity: refetch antrian on WebSocket events
    effect(() => {
      const updateTime = this.lastUpdateAt();
      if (updateTime) {
        this.fetchAntrian();
      }
    });

    // Refetch upon reconnected
    effect(() => {
      const status = this.connectionStatus();
      if (status === 'connected') {
        this.fetchAntrian();
      }
    });
  }

  ngOnInit(): void {
    // 1. Initialize live clock
    this.updateClock();
    const clockTimer = setInterval(() => this.updateClock(), 1000);
    this.destroyRef.onDestroy(() => clearInterval(clockTimer));

    // 2. Resolve display token from URL query params or localStorage
    const queryToken =
      this.route.snapshot.queryParamMap.get('token') ||
      this.route.snapshot.queryParamMap.get('displayToken');

    if (queryToken) {
      this.saveToken(queryToken);
    } else {
      const savedToken = this.loadToken();
      if (savedToken) {
        this.displayToken.set(savedToken);
      } else {
        this.showTokenModal.set(true);
      }
    }

    // 3. Initial fetch and realtime connect if token exists
    if (this.displayToken()) {
      this.initDisplayBoard();
    } else {
      this.isLoading.set(false);
    }

    // 4. Fallback polling every 30 seconds
    const fallbackPollTimer = setInterval(() => {
      if (this.displayToken()) {
        this.fetchAntrian();
      }
    }, 30000);

    this.destroyRef.onDestroy(() => {
      clearInterval(fallbackPollTimer);
      this.realtimeService.disconnect();
    });
  }

  initDisplayBoard(): void {
    const token = this.displayToken();
    if (!token) return;

    // Fetch clinic info & antrian using displayToken
    this.klinikService.fetchKlinikInfo(environment.defaultKlinikId, token).subscribe();
    this.fetchAntrian();
    this.realtimeService.connect({
      klinikId: environment.defaultKlinikId,
      displayToken: token,
    });
  }

  fetchAntrian(): void {
    const token = this.displayToken();
    if (!token) return;

    this.errorMessage.set(null);

    this.antrianService.getAntrian(environment.defaultKlinikId, token).subscribe({
      next: (data) => {
        // Sort: isPriority DESC, skipCount ASC, nomorAntrian ASC
        const sorted = [...data].sort((a, b) => {
          if (a.isPriority !== b.isPriority) {
            return a.isPriority ? -1 : 1;
          }
          const skipA = a.skipCount ?? 0;
          const skipB = b.skipCount ?? 0;
          if (skipA !== skipB) {
            return skipA - skipB;
          }
          return a.nomorAntrian - b.nomorAntrian;
        });

        this.antrianList.set(sorted);
        this.isLoading.set(false);
      },
      error: (err) => {
        this.isLoading.set(false);
        if (err?.status === 401 || err?.error?.code === 'UNAUTHORIZED') {
          this.errorMessage.set('Display Token tidak valid atau telah dicabut.');
          this.showTokenModal.set(true);
        } else {
          this.errorMessage.set('Gagal memuat antrian klinik. Mencoba kembali...');
        }
      },
    });
  }

  formatQueueNumber(num?: number | null): string {
    if (num === undefined || num === null) return '---';
    return num.toString().padStart(3, '0');
  }

  openTokenSettings(): void {
    this.tokenInput.set(this.displayToken());
    this.showTokenModal.set(true);
  }

  saveTokenFromModal(): void {
    const token = this.tokenInput().trim();
    if (!token) return;

    this.saveToken(token);
    this.showTokenModal.set(false);
    this.errorMessage.set(null);
    this.initDisplayBoard();
  }

  private saveToken(token: string): void {
    this.displayToken.set(token);
    if (typeof window !== 'undefined' && window.localStorage) {
      localStorage.setItem(DISPLAY_TOKEN_STORAGE_KEY, token);
    }
  }

  private loadToken(): string | null {
    if (typeof window !== 'undefined' && window.localStorage) {
      return localStorage.getItem(DISPLAY_TOKEN_STORAGE_KEY);
    }
    return null;
  }

  private updateClock(): void {
    const now = new Date();
    this.currentDayDateString.set(formatJakartaDayDate(now));
    this.currentTimeString.set(`${getJakartaTimeString(now)} WIB`);
  }
}
