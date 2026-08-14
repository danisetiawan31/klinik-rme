import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';

@Component({
  selector: 'app-toast',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (message()) {
      <div
        class="kl-toast-enter fixed top-5 left-1/2 -translate-x-1/2 z-50 w-full max-w-[420px] px-4"
      >
        <div
          [attr.role]="type() === 'error' ? 'alert' : 'status'"
          [attr.aria-live]="type() === 'error' ? 'assertive' : 'polite'"
          class="flex items-center gap-2.5 px-3.5 py-2.5 bg-card rounded-md shadow-3 border"
          [class.border-destructive]="type() === 'error'"
          [class.border-accent]="type() === 'success'"
          [class.border-border]="type() === 'info'"
        >
          <!-- Icon -->
          <span
            class="shrink-0 w-5 h-5 rounded-full flex items-center justify-center text-white"
            [class.bg-destructive]="type() === 'error'"
            [class.bg-accent]="type() === 'success'"
            [class.bg-primary]="type() === 'info'"
            aria-hidden="true"
          >
            @if (type() === 'error') {
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="10"
                height="10"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
                stroke-linecap="round"
              >
                <line x1="12" y1="5" x2="12" y2="13" />
                <circle cx="12" cy="18" r="1.2" fill="currentColor" stroke="none" />
              </svg>
            } @else if (type() === 'success') {
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="10"
                height="10"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
            } @else {
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="10"
                height="10"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
                stroke-linecap="round"
              >
                <circle cx="12" cy="8" r="1" fill="currentColor" stroke="none" />
                <line x1="12" y1="12" x2="12" y2="18" />
              </svg>
            }
          </span>

          <!-- Message -->
          <span
            class="flex-1 font-sans text-xs font-medium"
            [class.text-destructive]="type() === 'error'"
            [class.text-accent]="type() === 'success'"
            [class.text-foreground]="type() === 'info'"
            >{{ message() }}</span
          >

          <!-- Dismiss -->
          <button
            type="button"
            (click)="dismiss.emit()"
            aria-label="Tutup notifikasi"
            class="shrink-0 flex items-center justify-center w-6 h-6 bg-transparent border-0 cursor-pointer text-muted-foreground hover:text-foreground rounded-sm p-0 outline-none transition-colors"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              aria-hidden="true"
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
      </div>
    }
  `,
})
export class ToastComponent {
  message = input<string>('');
  type = input<'error' | 'success' | 'info'>('error');
  dismiss = output<void>();
}
