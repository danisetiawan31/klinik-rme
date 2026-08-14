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
  templateUrl: './pagination.component.html',
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
