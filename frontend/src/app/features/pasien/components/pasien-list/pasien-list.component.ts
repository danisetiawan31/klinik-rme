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
import { ToastComponent } from '../../../../shared/components/toast/toast.component';
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
    ToastComponent,
  ],
  template: `
    @if (errorMessage()) {
      <app-toast
        [message]="errorMessage() || ''"
        type="error"
        (dismiss)="errorMessage.set(null)"
      />
    }

    <!-- ── Page Wrapper (Zona Content — DESIGN.md §1.1) ── -->
    <div
      style="
        min-height: 100%;
        padding: var(--space-6);
        background-color: var(--color-background);
      "
    >
      <!-- Header section -->
      <div
        style="
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: var(--space-6);
        "
      >
        <div>
          <h1
            style="
              font-family: var(--font-heading);
              font-size: var(--text-2xl);
              font-weight: 700;
              color: var(--color-foreground);
              margin-bottom: var(--space-1);
            "
          >
            Pencarian &amp; Data Pasien
          </h1>
          <p
            style="
              font-family: var(--font-body);
              font-size: var(--text-sm);
              color: var(--color-muted-foreground);
            "
          >
            Cari data pasien berdasarkan Nama atau NIK.
          </p>
        </div>

        <a
          routerLink="/pasien/baru"
          class="kl-btn-primary"
          style="
            text-decoration: none;
            display: inline-flex;
            align-items: center;
            gap: var(--space-2);
          "
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16" height="16" viewBox="0 0 24 24"
            fill="none" stroke="currentColor" stroke-width="2.5"
            stroke-linecap="round" stroke-linejoin="round"
            aria-hidden="true"
          >
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          Registrasi Pasien Baru
        </a>
      </div>

      <!-- Search form card -->
      <div
        style="
          background: var(--color-card);
          border: 1px solid var(--color-border);
          border-radius: var(--radius-md);
          box-shadow: var(--shadow-2);
          padding: var(--space-6);
          margin-bottom: var(--space-6);
        "
      >
        <form
          [formGroup]="searchForm"
          style="
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
            gap: var(--space-4);
          "
        >
          <!-- Nama field (Debounced live-search) -->
          <div style="display:flex; flex-direction:column; gap:5px;">
            <label
              for="search-nama"
              style="
                font-family: var(--font-body);
                font-size: var(--text-sm);
                font-weight: 600;
                color: var(--color-foreground);
              "
            >
              Nama Pasien
            </label>
            <input
              id="search-nama"
              type="text"
              formControlName="nama"
              placeholder="Cari berdasarkan nama (min. 1 huruf)..."
              class="kl-input"
            />
          </div>

          <!-- NIK field (Auto-trigger exactly at 16 digits) -->
          <div style="display:flex; flex-direction:column; gap:5px;">
            <label
              for="search-nik"
              style="
                font-family: var(--font-body);
                font-size: var(--text-sm);
                font-weight: 600;
                color: var(--color-foreground);
              "
            >
              NIK
            </label>
            <input
              id="search-nik"
              type="text"
              formControlName="nik"
              inputmode="numeric"
              maxlength="16"
              placeholder="16 digit NIK..."
              class="kl-input"
            />
          </div>
        </form>
      </div>

      <!-- Results section -->
      <div
        style="
          background: var(--color-card);
          border: 1px solid var(--color-border);
          border-radius: var(--radius-md);
          box-shadow: var(--shadow-2);
          overflow: hidden;
        "
      >
        @if (isLoading()) {
          <div
            style="
              padding: var(--space-12);
              text-align: center;
              color: var(--color-muted-foreground);
              font-family: var(--font-body);
            "
          >
            <svg
              class="kl-spinner"
              xmlns="http://www.w3.org/2000/svg"
              width="24" height="24" viewBox="0 0 24 24"
              fill="none" stroke="var(--color-primary)" stroke-width="2.5"
              stroke-linecap="round" aria-hidden="true"
              style="margin: 0 auto 12px; display:block;"
            >
              <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
            </svg>
            Memuat data pasien...
          </div>
        } @else if (items().length === 0) {
          <!-- Empty State -->
          <div
            style="
              padding: var(--space-12);
              text-align: center;
              color: var(--color-muted-foreground);
              font-family: var(--font-body);
            "
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="40" height="40" viewBox="0 0 24 24"
              fill="none" stroke="currentColor" stroke-width="1.5"
              stroke-linecap="round" stroke-linejoin="round"
              style="margin: 0 auto 12px; display:block; opacity: 0.6;"
              aria-hidden="true"
            >
              <circle cx="11" cy="11" r="8"/>
              <line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            <p style="font-size: var(--text-base); font-weight: 600; color: var(--color-foreground); margin-bottom: 4px;">
              Tidak ada data pasien ditemukan
            </p>
            <p style="font-size: var(--text-sm); max-width: 360px; margin: 0 auto;">
              Coba ubah kata kunci pencarian NIK atau nama pasien di atas, atau daftarkan pasien baru.
            </p>
          </div>
        } @else {
          <!-- Table list -->
          <div style="overflow-x: auto;">
            <table
              style="
                width: 100%;
                border-collapse: collapse;
                text-align: left;
                font-family: var(--font-body);
                font-size: var(--text-sm);
              "
            >
              <thead>
                <tr
                  style="
                    background: var(--color-muted);
                    color: var(--color-foreground);
                    border-bottom: 1px solid var(--color-border);
                  "
                >
                  <th style="padding: 12px 16px; font-weight: 600;">NIK</th>
                  <th style="padding: 12px 16px; font-weight: 600;">Nama Lengkap</th>
                  <th style="padding: 12px 16px; font-weight: 600;">Tanggal Lahir</th>
                  <th style="padding: 12px 16px; font-weight: 600; text-align: right;">Aksi</th>
                </tr>
              </thead>
              <tbody>
                @for (pasien of items(); track pasien.id) {
                  <tr
                    style="
                      border-bottom: 1px solid var(--color-border);
                      transition: background 150ms;
                      cursor: pointer;
                    "
                    onmouseenter="this.style.background='var(--color-background)'"
                    onmouseleave="this.style.background='transparent'"
                    (click)="onSelectPasien(pasien.id)"
                  >
                    <td style="padding: 12px 16px;">
                      @if (pasien.nik) {
                        <app-sensitive-value
                          mode="display"
                          [displayValue]="pasien.nik"
                        />
                      } @else {
                        <span style="color: var(--color-muted-foreground); font-style: italic;">
                          (Tidak ada NIK)
                        </span>
                      }
                    </td>
                    <td style="padding: 12px 16px; font-weight: 600; color: var(--color-foreground);">
                      {{ pasien.nama }}
                    </td>
                    <td style="padding: 12px 16px; color: var(--color-muted-foreground);">
                      {{ formatDate(pasien.tanggalLahir) }}
                    </td>
                    <td style="padding: 12px 16px; text-align: right;">
                      <a
                        [routerLink]="['/pasien', pasien.id]"
                        class="kl-btn-secondary"
                        style="padding: 4px 10px; font-size: var(--text-xs); text-decoration: none;"
                        (click)="$event.stopPropagation()"
                      >
                        Detail &rarr;
                      </a>
                    </td>
                  </tr>
                }
              </tbody>
            </table>
          </div>

          <!-- Pagination -->
          <app-pagination
            [page]="page()"
            [limit]="limit()"
            [totalCount]="totalCount()"
            (pageChange)="onPageChange($event)"
          />
        }
      </div>
    </div>
  `,
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
