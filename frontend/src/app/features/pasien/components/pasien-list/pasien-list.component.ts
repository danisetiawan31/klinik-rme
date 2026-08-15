import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { debounceTime, distinctUntilChanged } from 'rxjs';
import { PaginationComponent } from '../../../../shared/components/pagination/pagination.component';
import { SensitiveValueComponent } from '../../../../shared/components/sensitive-value/sensitive-value.component';
import { HlmButton } from '../../../../shared/ui/button/src/lib/hlm-button';
import { HlmSkeletonImports } from '../../../../shared/ui/skeleton/src/index';
import { HlmTableImports } from '../../../../shared/ui/table/src/index';
import { PasienService } from '../../pasien.service';
import { PasienSearchItem } from '../../pasien.types';

@Component({
  selector: 'app-pasien-list',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    RouterLink,
    PaginationComponent,
    SensitiveValueComponent,
    HlmButton,
    ...HlmSkeletonImports,
    ...HlmTableImports,
  ],
  templateUrl: './pasien-list.component.html',
})
export class PasienListComponent implements OnInit {
  private fb = inject(FormBuilder);
  private pasienService = inject(PasienService);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);

  readonly items = signal<PasienSearchItem[]>([]);
  readonly totalCount = signal<number>(0);
  readonly page = signal<number>(1);
  readonly limit = signal<number>(10);

  readonly isLoading = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  readonly searchForm = this.fb.group({
    nama: [''],
    nik: [''],
  });

  ngOnInit(): void {
    // Initial fetch
    this.fetchData();

    // 1) Nama search listener: Debounce 300ms, resets page to 1
    this.searchForm.controls.nama.valueChanges
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        takeUntilDestroyed(this.destroyRef)
      )
      .subscribe(() => {
        this.page.set(1);
        this.fetchData();
      });

    // 2) NIK search listener: Auto-trigger search ONLY when exactly 16 digits
    this.searchForm.controls.nik.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((val) => {
        const trimmed = val?.trim() || '';
        // Fire search if empty (to clear NIK filter) or if exactly 16 digits
        if (trimmed === '' || /^\d{16}$/.test(trimmed)) {
          this.page.set(1);
          this.fetchData();
        }
      });
  }

  onPageChange(newPage: number): void {
    this.page.set(newPage);
    this.fetchData();
  }

  fetchData(): void {
    this.isLoading.set(true);
    this.errorMessage.set(null);

    const { nama, nik } = this.searchForm.getRawValue();

    this.pasienService
      .search({
        nama: nama || undefined,
        nik: nik || undefined,
        page: this.page(),
        limit: this.limit(),
      })
      .subscribe({
        next: (result) => {
          this.isLoading.set(false);
          this.items.set(result.items);
          this.totalCount.set(result.totalCount);
        },
        error: (err: any) => {
          this.isLoading.set(false);
          const msg =
            err?.error?.error?.message ??
            'Gagal memuat data pasien. Silakan coba lagi.';
          this.errorMessage.set(msg);
        },
      });
  }

  onSelectPasien(id: number): void {
    this.router.navigate(['/pasien', id]);
  }

  formatDate(isoDateStr: string): string {
    if (!isoDateStr) return '-';
    try {
      const d = new Date(isoDateStr);
      return d.toLocaleDateString('id-ID', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
      });
    } catch {
      return isoDateStr;
    }
  }
}
