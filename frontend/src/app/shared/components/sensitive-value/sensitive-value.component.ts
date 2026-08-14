import {
  ChangeDetectionStrategy,
  Component,
  forwardRef,
  input,
  signal,
} from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';

@Component({
  selector: 'app-sensitive-value',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => SensitiveValueComponent),
      multi: true,
    },
  ],
  template: `
    @if (mode() === 'input') {
      <div class="kl-pw-wrap">
        <input
          [type]="isRevealed() ? 'text' : 'password'"
          [placeholder]="placeholder()"
          [disabled]="isDisabled()"
          [value]="value()"
          (input)="onInput($event)"
          (blur)="onBlur()"
          class="kl-input"
          style="padding-right:40px;"
          autocomplete="current-password"
          [attr.aria-label]="placeholder() || 'Kata sandi'"
        />
        <button
          type="button"
          class="kl-pw-toggle"
          (click)="toggleReveal()"
          [disabled]="isDisabled()"
          [attr.aria-label]="isRevealed() ? 'Sembunyikan password' : 'Tampilkan password'"
          [attr.aria-pressed]="isRevealed()"
        >
          @if (isRevealed()) {
            <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17"
              viewBox="0 0 24 24" fill="none" stroke="currentColor"
              stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              aria-hidden="true">
              <path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/>
              <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"/>
              <path d="M6.61 6.61A13.52 13.52 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"/>
              <line x1="2" y1="2" x2="22" y2="22"/>
            </svg>
          } @else {
            <svg xmlns="http://www.w3.org/2000/svg" width="17" height="17"
              viewBox="0 0 24 24" fill="none" stroke="currentColor"
              stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              aria-hidden="true">
              <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
          }
        </button>
      </div>
    } @else {
      <!-- Display mode: NIK masking -->
      <span class="font-mono text-sm inline-flex items-center gap-1.5">
        {{ formattedDisplayValue() }}
        <button
          type="button"
          class="kl-pw-toggle"
          style="position:static;transform:none;width:22px;height:22px;"
          (click)="toggleReveal()"
          [attr.aria-label]="isRevealed() ? 'Sembunyikan NIK' : 'Tampilkan NIK'"
          [attr.aria-pressed]="isRevealed()"
        >
          @if (isRevealed()) {
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14"
              viewBox="0 0 24 24" fill="none" stroke="currentColor"
              stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              aria-hidden="true">
              <path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"/>
              <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"/>
              <path d="M6.61 6.61A13.52 13.52 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"/>
              <line x1="2" y1="2" x2="22" y2="22"/>
            </svg>
          } @else {
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14"
              viewBox="0 0 24 24" fill="none" stroke="currentColor"
              stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
              aria-hidden="true">
              <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
          }
        </button>
      </span>
    }
  `,
})
export class SensitiveValueComponent implements ControlValueAccessor {
  mode = input<'input' | 'display'>('input');
  placeholder = input<string>('');
  id = input<string>('');
  name = input<string>('');
  displayValue = input<string>('');

  readonly isRevealed = signal<boolean>(false);
  readonly value = signal<string>('');
  readonly isDisabled = signal<boolean>(false);

  private onChange: (val: string) => void = () => {};
  private onTouched: () => void = () => {};

  toggleReveal(): void {
    if (this.isDisabled()) return;
    this.isRevealed.update((v) => !v);
  }

  onInput(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    this.value.set(val);
    this.onChange(val);
  }

  onBlur(): void {
    this.onTouched();
  }

  formattedDisplayValue(): string {
    const raw = this.displayValue() || this.value();
    if (!raw) return '-';
    if (this.isRevealed()) return raw;
    if (raw.length <= 4) return '••••' + raw;
    return '•'.repeat(raw.length - 4) + raw.slice(-4);
  }

  writeValue(val: string): void { this.value.set(val || ''); }
  registerOnChange(fn: (val: string) => void): void { this.onChange = fn; }
  registerOnTouched(fn: () => void): void { this.onTouched = fn; }
  setDisabledState(disabled: boolean): void { this.isDisabled.set(disabled); }
}
