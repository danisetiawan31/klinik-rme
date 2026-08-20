import {
  ChangeDetectionStrategy,
  Component,
  input,
  output,
} from '@angular/core';
import { FormArray, ReactiveFormsModule } from '@angular/forms';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucidePill, lucidePlus, lucideTrash2 } from '@ng-icons/lucide';
import { HlmButton } from '../../../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../../../../../shared/ui/input/src/lib/hlm-input';

@Component({
  selector: 'app-soap-plan-section',
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
      lucidePill,
    }),
  ],
  templateUrl: './soap-plan-section.component.html',
})
export class SoapPlanSectionComponent {
  readonly tindakanArray = input.required<FormArray>();

  readonly addTindakan = output<'resep' | 'tindakan'>();
  readonly removeTindakan = output<number>();

  onAdd(jenis: 'resep' | 'tindakan'): void {
    this.addTindakan.emit(jenis);
  }

  onRemove(index: number): void {
    this.removeTindakan.emit(index);
  }
}
