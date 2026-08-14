import {
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  output,
} from '@angular/core';

@Component({
  selector: 'app-pagination',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div
      class="flex items-center justify-between px-4 py-3 bg-card border-t border-border font-sans text-sm text-muted-foreground"
    >
      <!-- Page counter info -->
      <div>
        <span>Halaman <strong class="text-foreground">{{ page() }}</strong> dari <strong class="text-foreground">{{ totalPages() }}</strong></span>
        @if (totalCount() > 0) {
          <span class="ml-2 text-muted-foreground">
            ({{ totalCount() }} total record)
          </span>
        }
      </div>

      <!-- Navigation buttons -->
      <div class="flex gap-2">
        <button
          type="button"
          class="px-3 py-1 text-xs border border-border text-foreground hover:bg-muted font-medium rounded-sm disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          [disabled]="isPrevDisabled()"
          (click)="onPrev()"
          aria-label="Halaman sebelumnya"
        >
          &larr; Sebelumnya
        </button>
        <button
          type="button"
          class="px-3 py-1 text-xs border border-border text-foreground hover:bg-muted font-medium rounded-sm disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          [disabled]="isNextDisabled()"
          (click)="onNext()"
          aria-label="Halaman selanjutnya"
        >
          Selanjutnya &rarr;
        </button>
      </div>
    </div>
  `,
})
export class PaginationComponent {
  page = input<number>(1);
  limit = input<number>(10);
  totalCount = input<number>(0);

  pageChange = output<number>();

  readonly totalPages = computed(() => {
    const total = this.totalCount();
    const lim = this.limit();
    if (!lim || lim <= 0) return 1;
    return Math.max(1, Math.ceil(total / lim));
  });

  readonly isPrevDisabled = computed(() => this.page() <= 1);
  readonly isNextDisabled = computed(() => this.page() >= this.totalPages());

  onPrev(): void {
    if (!this.isPrevDisabled()) {
      this.pageChange.emit(this.page() - 1);
    }
  }

  onNext(): void {
    if (!this.isNextDisabled()) {
      this.pageChange.emit(this.page() + 1);
    }
  }
}
