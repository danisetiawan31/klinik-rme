import {
  ChangeDetectionStrategy,
  Component,
  input,
  output,
} from '@angular/core';
import { FormArray, ReactiveFormsModule } from '@angular/forms';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucidePlus, lucideTrash2 } from '@ng-icons/lucide';
import { HlmButton } from '../../../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../../../../../shared/ui/input/src/lib/hlm-input';

@Component({
  selector: 'app-soap-assessment-section',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    HlmButton,
    HlmInput,
    NgIcon,
    HlmIconDirective,
    ...HlmCardImports,
  ],
  providers: [
    provideIcons({
      lucidePlus,
      lucideTrash2,
    }),
  ],
  templateUrl: './soap-assessment-section.component.html',
})
export class SoapAssessmentSectionComponent {
  readonly diagnosisArray = input.required<FormArray>();

  readonly addDiagnosis = output<void>();
  readonly removeDiagnosis = output<number>();

  onAdd(): void {
    this.addDiagnosis.emit();
  }

  onRemove(index: number): void {
    this.removeDiagnosis.emit(index);
  }
}
