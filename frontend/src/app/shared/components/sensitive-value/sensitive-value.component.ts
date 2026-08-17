import {
  ChangeDetectionStrategy,
  Component,
  forwardRef,
  input,
  signal,
} from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucideEye, lucideEyeOff } from '@ng-icons/lucide';
import { HlmIconDirective } from '../../ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../ui/input/src/lib/hlm-input';

@Component({
  selector: 'app-sensitive-value',
  standalone: true,
  imports: [HlmInput, NgIcon, HlmIconDirective],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => SensitiveValueComponent),
      multi: true,
    },
    provideIcons({ lucideEye, lucideEyeOff }),
  ],
  templateUrl: './sensitive-value.component.html',
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
