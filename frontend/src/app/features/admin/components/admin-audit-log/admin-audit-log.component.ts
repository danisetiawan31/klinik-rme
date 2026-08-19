import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideAlertCircle,
  lucideClock,
  lucideEye,
  lucideFileSpreadsheet,
  lucideFileText,
  lucideFilter,
  lucideRotateCcw,
  lucideSearch,
  lucideShieldAlert,
  lucideUser,
} from '@ng-icons/lucide';
import { toast } from '@spartan-ng/brain/sonner';
import { formatJakartaDate, getJakartaTimeString } from '../../../../core/utils/date.utils';
import { PaginationComponent } from '../../../../shared/components/pagination/pagination.component';
import { HlmAlertImports } from '../../../../shared/ui/alert/src/index';
import { HlmBadge } from '../../../../shared/ui/badge/src/lib/hlm-badge';
import { HlmButton } from '../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../../../shared/ui/input/src/lib/hlm-input';
import { HlmLabel } from '../../../../shared/ui/label/src/lib/hlm-label';
import { HlmSkeletonImports } from '../../../../shared/ui/skeleton/src/index';
import { HlmTableImports } from '../../../../shared/ui/table/src/index';
import { AdminService } from '../../admin.service';
import { AuditLogDetail, AuditLogFilterParams, AuditLogSummary } from '../../admin.types';
import { AuditDiffViewerComponent } from '../audit-diff-viewer/audit-diff-viewer.component';

@Component({
  selector: 'app-admin-audit-log',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    PaginationComponent,
    AuditDiffViewerComponent,
    HlmButton,
    HlmInput,
    HlmLabel,
    HlmBadge,
    NgIcon,
    HlmIconDirective,
    ...HlmAlertImports,
    ...HlmCardImports,
    ...HlmSkeletonImports,
    ...HlmTableImports,
  ],
  providers: [
    provideIcons({
      lucideFileText,
      lucideFilter,
      lucideRotateCcw,
      lucideSearch,
      lucideEye,
      lucideClock,
      lucideUser,
      lucideAlertCircle,
      lucideShieldAlert,
      lucideFileSpreadsheet,
    }),
  ],
  templateUrl: './admin-audit-log.component.html',
})
export class AdminAuditLogComponent implements OnInit {
  private adminService = inject(AdminService);
  private fb = inject(FormBuilder);

  // Table & pagination state
  readonly logs = signal<AuditLogSummary[]>([]);
  readonly totalCount = signal<number>(0);
  readonly currentPage = signal<number>(1);
  readonly limit = 10;
  readonly isLoading = signal<boolean>(true);
  readonly errorMessage = signal<string | null>(null);

  // Filter Form
  readonly filterForm: FormGroup = this.fb.group({
    tabelTarget: [''],
    recordId: [''],
    actorId: [''],
  });

  // Detail Modal state
  readonly selectedLogDetail = signal<AuditLogDetail | null>(null);
  readonly isLoadingDetail = signal<boolean>(false);

  ngOnInit(): void {
    this.loadLogs(1);
  }

  loadLogs(page: number): void {
    this.isLoading.set(true);
    this.errorMessage.set(null);
    this.currentPage.set(page);

    const fv = this.filterForm.value;
    const params: AuditLogFilterParams = {
      page,
      limit: this.limit,
      tabelTarget: fv.tabelTarget || undefined,
      recordId: fv.recordId ? parseInt(fv.recordId, 10) : undefined,
      actorId: fv.actorId ? parseInt(fv.actorId, 10) : undefined,
    };

    this.adminService.getAuditLogs(params).subscribe({
      next: (res) => {
        this.logs.set(res.logs);
        this.totalCount.set(res.totalCount);
        this.isLoading.set(false);
      },
      error: (err) => {
        this.isLoading.set(false);
        this.errorMessage.set(err?.error?.message || 'Gagal memuat log audit.');
      },
    });
  }

  applyFilter(): void {
    this.loadLogs(1);
  }

  resetFilter(): void {
    this.filterForm.reset({
      tabelTarget: '',
      recordId: '',
      actorId: '',
    });
    this.loadLogs(1);
  }

  openDetail(logId: number): void {
    this.isLoadingDetail.set(true);

    this.adminService.getAuditLogDetail(logId).subscribe({
      next: (detail) => {
        this.isLoadingDetail.set(false);
        this.selectedLogDetail.set(detail);
      },
      error: (err) => {
        this.isLoadingDetail.set(false);
        toast.error(err?.error?.message || 'Gagal memuat detail log audit.');
      },
    });
  }

  closeDetail(): void {
    this.selectedLogDetail.set(null);
  }

  formatDate(dateStr: string): string {
    return `${formatJakartaDate(dateStr)} ${getJakartaTimeString(dateStr)}`;
  }
}
