import {
  ChangeDetectionStrategy,
  Component,
  effect,
  input,
  output,
  untracked,
} from '@angular/core';
import { toast } from '@spartan-ng/brain/sonner';

@Component({
  selector: 'app-toast',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './toast.component.html',
})
export class ToastComponent {
  message = input<string>('');
  type = input<'error' | 'success' | 'info'>('error');
  dismiss = output<void>();

  constructor() {
    effect(() => {
      const msg = this.message();
      const t = this.type();
      if (!msg) return;

      untracked(() => {
        if (t === 'error') {
          toast.error(msg, { onDismiss: () => this.dismiss.emit() });
        } else if (t === 'success') {
          toast.success(msg, { onDismiss: () => this.dismiss.emit() });
        } else {
          toast.info(msg, { onDismiss: () => this.dismiss.emit() });
        }
      });
    });
  }
}
