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
import { RouterLink } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideActivity,
  lucideArrowRight,
  lucideCheckCircle2,
  lucideClock,
  lucideFileEdit,
  lucideFileText,
  lucideListOrdered,
  lucidePlus,
  lucideSearch,
  lucideStethoscope,
  lucideUserCheck,
  lucideUsers,
} from '@ng-icons/lucide';
import { of, Subject } from 'rxjs';
import { debounceTime, distinctUntilChanged, switchMap } from 'rxjs/operators';
import { AuthService } from '../../../../core/auth/auth.service';
import { RealtimeService } from '../../../../core/realtime/realtime.service';
import { PriorityBadgeComponent } from '../../../../shared/components/priority-badge/priority-badge.component';
import { StatusBadgeComponent } from '../../../../shared/components/status-badge/status-badge.component';
import { HlmButton } from '../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../shared/ui/card/src/index';
import { HlmEmptyImports } from '../../../../shared/ui/empty/src/lib/hlm-empty';
import { HlmIconDirective } from '../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../../../shared/ui/input/src/lib/hlm-input';
import { HlmSkeletonImports } from '../../../../shared/ui/skeleton/src/index';
import { AntrianService } from '../../../antrian/antrian.service';
import { KunjunganListItem } from '../../../antrian/antrian.types';
import { PasienService } from '../../../pasien/pasien.service';
import { PasienSearchItem } from '../../../pasien/pasien.types';

@Component({
  selector: 'app-rekam-medis-list',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FormsModule,
    RouterLink,
    NgIcon,
    HlmIconDirective,
    HlmButton,
    HlmInput,
    StatusBadgeComponent,
    PriorityBadgeComponent,
    ...HlmCardImports,
    HlmSkeletonImports,
    HlmEmptyImports,
  ],
  providers: [
    provideIcons({
      lucideStethoscope,
      lucideFileText,
      lucideSearch,
      lucideActivity,
      lucideCheckCircle2,
      lucideClock,
      lucideArrowRight,
      lucideFileEdit,
      lucideUsers,
      lucideListOrdered,
      lucidePlus,
      lucideUserCheck,
    }),
  ],
  templateUrl: './rekam-medis-list.component.html',
})
export class RekamMedisListComponent implements OnInit {
  private antrianService = inject(AntrianService);
  private pasienService = inject(PasienService);
  private realtimeService = inject(RealtimeService);
  readonly authService = inject(AuthService);
  private destroyRef = inject(DestroyRef);

  // State Signals
  readonly antrianList = signal<KunjunganListItem[]>([]);
  readonly isLoadingAntrian = signal<boolean>(true);
  readonly activeTab = signal<'dipanggil' | 'selesai' | 'semua'>('dipanggil');

  // Search State
  readonly searchQuery = signal<string>('');
  readonly searchResults = signal<PasienSearchItem[]>([]);
  readonly isSearching = signal<boolean>(false);
  private searchSubject = new Subject<string>();

  // Active calling patient (status: 'dipanggil')
  readonly activeCallingPatient = computed(() => {
    return this.antrianList().find((item) => item.status === 'dipanggil') || null;
  });

  // Filtered visits
  readonly filteredList = computed(() => {
    const list = this.antrianList();
    const tab = this.activeTab();
    if (tab === 'dipanggil') {
      return list.filter((item) => item.status === 'dipanggil');
    }
    if (tab === 'selesai') {
      return list.filter((item) => item.status === 'selesai');
    }
    return list;
  });

  readonly totalSelesai = computed(() => {
    return this.antrianList().filter((item) => item.status === 'selesai').length;
  });

  readonly totalDipanggil = computed(() => {
    return this.antrianList().filter((item) => item.status === 'dipanggil').length;
  });

  readonly totalMenunggu = computed(() => {
    return this.antrianList().filter((item) => item.status === 'menunggu').length;
  });

  constructor() {
    // Re-fetch when realtime signal updates
    effect(() => {
      const updateAt = this.realtimeService.lastUpdateAt();
      if (updateAt) {
        this.loadAntrian();
      }
    });
  }

  ngOnInit(): void {
    this.loadAntrian();

    // Debounced search reactive pipeline with switchMap to prevent race conditions
    const searchSub = this.searchSubject
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        switchMap((query) => {
          const trimmed = query.trim();
          if (!trimmed) {
            this.isSearching.set(false);
            this.searchResults.set([]);
            return of({ items: [], totalCount: 0 });
          }
          this.isSearching.set(true);
          const isNum = /^\d+$/.test(trimmed);
          const params = isNum
            ? { nik: trimmed, page: 1, limit: 5 }
            : { nama: trimmed, page: 1, limit: 5 };
          return this.pasienService.search(params);
        })
      )
      .subscribe({
        next: (res) => {
          this.searchResults.set(res.items || []);
          this.isSearching.set(false);
        },
        error: () => {
          this.searchResults.set([]);
          this.isSearching.set(false);
        },
      });

    this.destroyRef.onDestroy(() => {
      searchSub.unsubscribe();
    });
  }

  loadAntrian(): void {
    this.isLoadingAntrian.set(true);
    this.antrianService.getAntrian().subscribe({
      next: (data) => {
        this.antrianList.set(data || []);
        this.isLoadingAntrian.set(false);

        // Auto-switch tab if no active calling but completed exists
        const hasCalling = data.some((d) => d.status === 'dipanggil');
        if (!hasCalling && this.activeTab() === 'dipanggil') {
          this.activeTab.set('selesai');
        }
      },
      error: () => {
        this.isLoadingAntrian.set(false);
      },
    });
  }

  onSearchInput(value: string): void {
    this.searchQuery.set(value);
    this.searchSubject.next(value);
  }

  formatQueueNumber(num?: number | null): string {
    if (num === undefined || num === null) return '---';
    return num.toString().padStart(3, '0');
  }
}
