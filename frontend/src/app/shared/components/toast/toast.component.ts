import {
  ChangeDetectionStrategy,
  Component,
  input,
  output,
} from '@angular/core';

@Component({
  selector: 'app-toast',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (message()) {
      <div
        class="kl-toast-enter"
        style="
          position:fixed;
          top:20px;
          left:50%;
          transform:translateX(-50%);
          z-index:50;
          width:100%;
          max-width:420px;
          padding:0 16px;
        "
      >
        <div
          [attr.role]="type() === 'error' ? 'alert' : 'status'"
          [attr.aria-live]="type() === 'error' ? 'assertive' : 'polite'"
          style="
            display:flex;
            align-items:center;
            gap:10px;
            padding:11px 14px;
            background:#FFFFFF;
            border-radius:8px;
            box-shadow:0 8px 24px rgba(0,0,0,0.10), 0 0 0 1px rgba(0,0,0,0.05);
          "
          [style.border]="type() === 'error' ? '1.5px solid #FCA5A5' :
                          type() === 'success' ? '1.5px solid #86EFAC' : '1.5px solid #CCFBF1'"
        >
          <!-- Icon -->
          <span
            style="flex-shrink:0;width:20px;height:20px;border-radius:50%;display:flex;align-items:center;justify-content:center;"
            [style.background]="type() === 'error' ? '#DC2626' : type() === 'success' ? '#16A34A' : '#0891B2'"
            aria-hidden="true"
          >
            @if (type() === 'error') {
              <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"
                viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round">
                <line x1="12" y1="5" x2="12" y2="13"/>
                <circle cx="12" cy="18" r="1.2" fill="#fff" stroke="none"/>
              </svg>
            } @else if (type() === 'success') {
              <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"
                viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3"
                stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
            } @else {
              <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"
                viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round">
                <circle cx="12" cy="8" r="1" fill="#fff" stroke="none"/>
                <line x1="12" y1="12" x2="12" y2="18"/>
              </svg>
            }
          </span>

          <!-- Message -->
          <span
            style="flex:1;font-family:var(--font-body);font-size:13px;font-weight:500;"
            [style.color]="type() === 'error' ? '#DC2626' : type() === 'success' ? '#15803D' : '#134E4A'"
          >{{ message() }}</span>

          <!-- Dismiss -->
          <button
            type="button"
            (click)="dismiss.emit()"
            aria-label="Tutup notifikasi"
            style="
              flex-shrink:0;display:flex;align-items:center;justify-content:center;
              width:24px;height:24px;background:transparent;border:none;cursor:pointer;
              color:#94A3B8;border-radius:4px;padding:0;outline:none;
              transition:color 150ms;
            "
            onmouseenter="this.style.color='#475569'"
            onmouseleave="this.style.color='#94A3B8'"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14"
              viewBox="0 0 24 24" fill="none" stroke="currentColor"
              stroke-width="2.5" stroke-linecap="round" aria-hidden="true">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
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
