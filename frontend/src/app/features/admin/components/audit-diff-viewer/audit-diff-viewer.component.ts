import {
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  output,
} from '@angular/core';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideAlertTriangle,
  lucideArrowRight,
  lucideCheck,
  lucideClock,
  lucideDatabase,
  lucideFileSpreadsheet,
  lucideFileText,
  lucideHash,
  lucideShieldAlert,
  lucideUser,
  lucideX,
} from '@ng-icons/lucide';
import { formatJakartaDate, getJakartaTimeString } from '../../../../core/utils/date.utils';
import { HlmAlertImports } from '../../../../shared/ui/alert/src/index';
import { HlmBadge } from '../../../../shared/ui/badge/src/lib/hlm-badge';
import { HlmButton } from '../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { AuditLogDetail } from '../../admin.types';

export interface DiffFieldEntry {
  key: string;
  beforeVal: any;
  afterVal: any;
  isModified: boolean;
  isAdded: boolean;
  isRemoved: boolean;
  beforeFormatted: string;
  afterFormatted: string;
}

@Component({
  selector: 'app-audit-diff-viewer',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    HlmButton,
    HlmBadge,
    NgIcon,
    HlmIconDirective,
    ...HlmAlertImports,
    ...HlmCardImports,
  ],
  providers: [
    provideIcons({
      lucideFileText,
      lucideClock,
      lucideUser,
      lucideDatabase,
      lucideHash,
      lucideShieldAlert,
      lucideAlertTriangle,
      lucideArrowRight,
      lucideCheck,
      lucideX,
      lucideFileSpreadsheet,
    }),
  ],
  templateUrl: './audit-diff-viewer.component.html',
})
export class AuditDiffViewerComponent {
  readonly log = input.required<AuditLogDetail>();
  readonly closed = output<void>();

  readonly formattedDate = computed(() => {
    const d = this.log()?.createdAt;
    if (!d) return '-';
    return `${formatJakartaDate(d)} ${getJakartaTimeString(d)}`;
  });

  readonly isRekamMedis = computed(() => {
    return this.log()?.tabelTarget === 'rekam_medis';
  });

  readonly diffEntries = computed<DiffFieldEntry[]>(() => {
    const before = this.log()?.beforeData || {};
    const after = this.log()?.afterData || {};

    const allKeys = Array.from(
      new Set([...Object.keys(before), ...Object.keys(after)])
    ).sort();

    return allKeys.map((key) => {
      const beforeVal = before[key];
      const afterVal = after[key];
      const isAdded = beforeVal === undefined && afterVal !== undefined;
      const isRemoved = beforeVal !== undefined && afterVal === undefined;
      const isModified =
        !isAdded &&
        !isRemoved &&
        JSON.stringify(beforeVal) !== JSON.stringify(afterVal);

      return {
        key,
        beforeVal,
        afterVal,
        isModified,
        isAdded,
        isRemoved,
        beforeFormatted: this.formatValue(beforeVal),
        afterFormatted: this.formatValue(afterVal),
      };
    });
  });

  private formatValue(val: any): string {
    if (val === undefined) return '(tidak ada)';
    if (val === null) return 'null';
    if (typeof val === 'object') {
      return JSON.stringify(val, null, 2);
    }
    return String(val);
  }

  onClose(): void {
    this.closed.emit();
  }
}
