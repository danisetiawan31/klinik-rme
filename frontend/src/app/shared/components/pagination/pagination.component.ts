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
      style="
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: var(--space-3) var(--space-4);
        background: var(--color-card);
        border-top: 1px solid var(--color-border);
        font-family: var(--font-body);
        font-size: var(--text-sm);
        color: var(--color-muted-foreground);
      "
    >
      <!-- Page counter info -->
      <div>
        <span>Halaman <strong>{{ page() }}</strong> dari <strong>{{ totalPages() }}</strong></span>
        @if (totalCount() > 0) {
          <span style="margin-left: var(--space-2); color: var(--color-muted-foreground);">
            ({{ totalCount() }} total record)
          </span>
        }
      </div>

      <!-- Navigation buttons -->
      <div style="display: flex; gap: var(--space-2);">
        <button
          type="button"
          class="kl-btn-secondary"
          style="padding: 4px 12px; font-size: var(--text-xs);"
          [disabled]="isPrevDisabled()"
          (click)="onPrev()"
          aria-label="Halaman sebelumnya"
        >
          &larr; Sebelumnya
        </button>
        <button
          type="button"
          class="kl-btn-secondary"
          style="padding: 4px 12px; font-size: var(--text-xs);"
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
