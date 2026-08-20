import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { FormGroup, ReactiveFormsModule } from '@angular/forms';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucideAlertCircle } from '@ng-icons/lucide';
import { HlmCardImports } from '../../../../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmLabel } from '../../../../../../shared/ui/label/src/lib/hlm-label';
import { HlmTextarea } from '../../../../../../shared/ui/textarea/src/lib/hlm-textarea';

@Component({
  selector: 'app-soap-subjective-section',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    HlmLabel,
    HlmTextarea,
    NgIcon,
    HlmIconDirective,
    ...HlmCardImports,
  ],
  providers: [
    provideIcons({
      lucideAlertCircle,
    }),
  ],
  templateUrl: './soap-subjective-section.component.html',
})
export class SoapSubjectiveSectionComponent {
  readonly form = input.required<FormGroup>();
}
