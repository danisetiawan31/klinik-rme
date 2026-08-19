import {
  ChangeDetectionStrategy,
  Component,
  input,
  output,
  signal,
} from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideAlertTriangle,
  lucideCheck,
  lucideCopy,
  lucideKey,
} from '@ng-icons/lucide';
import { toast } from '@spartan-ng/brain/sonner';
import { HlmAlertImports } from '../../ui/alert/src/index';
import { HlmButton } from '../../ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../ui/card/src/index';
import { HlmIconDirective } from '../../ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../ui/input/src/lib/hlm-input';
import { HlmLabel } from '../../ui/label/src/lib/hlm-label';

@Component({
  selector: 'app-reveal-once-secret',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    HlmButton,
    HlmInput,
    HlmLabel,
    NgIcon,
    HlmIconDirective,
    ...HlmCardImports,
    ...HlmAlertImports,
  ],
  providers: [
    provideIcons({
      lucideKey,
      lucideCopy,
      lucideCheck,
      lucideAlertTriangle,
    }),
  ],
  templateUrl: './reveal-once-secret.component.html',
})
export class RevealOnceSecretComponent {
  readonly title = input<string>('Rahasia Dibuat (Sekali Lihat)');
  readonly label = input<string>('Nilai Token / Tautan');
  readonly secretValue = input.required<string>();
  readonly description = input<string>(
    'Nilai rahasia ini hanya ditampilkan sekali pada sesi ini. Simpan atau salin sebelum menutup jendela.'
  );

  readonly closed = output<void>();

  readonly isCopied = signal<boolean>(false);

  copyToClipboard(): void {
    const val = this.secretValue();
    if (!val) return;

    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      navigator.clipboard.writeText(val).then(() => {
        this.isCopied.set(true);
        toast.success('Berhasil disalin ke clipboard!');
        setTimeout(() => this.isCopied.set(false), 2500);
      });
    }
  }

  onClose(): void {
    this.closed.emit();
  }
}
