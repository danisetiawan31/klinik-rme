import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { AdminService } from '../../admin.service';
import { AuditLogDetail, AuditLogSummary } from '../../admin.types';
import { AdminAuditLogComponent } from './admin-audit-log.component';

describe('AdminAuditLogComponent', () => {
  let component: AdminAuditLogComponent;
  let fixture: ComponentFixture<AdminAuditLogComponent>;
  let mockAdminService: {
    getAuditLogs: ReturnType<typeof vi.fn>;
    getAuditLogDetail: ReturnType<typeof vi.fn>;
  };

  const mockLogsList: AuditLogSummary[] = [
    {
      id: 10,
      tabelTarget: 'pasien',
      recordId: 5,
      actorUserId: 1,
      aksi: 'update',
      createdAt: '2026-08-19T10:00:00Z',
    },
    {
      id: 11,
      tabelTarget: 'rekam_medis',
      recordId: 3,
      actorUserId: 2,
      aksi: 'insert',
      createdAt: '2026-08-19T10:30:00Z',
    },
  ];

  beforeEach(async () => {
    mockAdminService = {
      getAuditLogs: vi.fn().mockReturnValue(of({ logs: mockLogsList, totalCount: 2 })),
      getAuditLogDetail: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [AdminAuditLogComponent],
      providers: [{ provide: AdminService, useValue: mockAdminService }],
    }).compileComponents();

    fixture = TestBed.createComponent(AdminAuditLogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and load audit logs on init', () => {
    expect(component).toBeTruthy();
    expect(mockAdminService.getAuditLogs).toHaveBeenCalledWith({
      page: 1,
      limit: 10,
      tabelTarget: undefined,
      recordId: undefined,
      actorId: undefined,
    });
    expect(component.logs()).toEqual(mockLogsList);
    expect(component.totalCount()).toBe(2);
    expect(component.isLoading()).toBe(false);
  });

  it('should apply filter and reload logs from page 1', () => {
    component.filterForm.patchValue({
      tabelTarget: 'rekam_medis',
      recordId: '3',
      actorId: '2',
    });

    component.applyFilter();

    expect(mockAdminService.getAuditLogs).toHaveBeenCalledWith({
      page: 1,
      limit: 10,
      tabelTarget: 'rekam_medis',
      recordId: 3,
      actorId: 2,
    });
  });

  it('should reset filter to empty and reload logs', () => {
    component.filterForm.patchValue({
      tabelTarget: 'pasien',
      recordId: '10',
    });

    component.resetFilter();

    expect(component.filterForm.value).toEqual({
      tabelTarget: '',
      recordId: '',
      actorId: '',
    });
    expect(mockAdminService.getAuditLogs).toHaveBeenCalled();
  });

  it('should open audit log detail modal when openDetail is called', () => {
    const mockDetail: AuditLogDetail = {
      id: 10,
      tabelTarget: 'pasien',
      recordId: 5,
      actorUserId: 1,
      aksi: 'update',
      beforeData: { nama: 'Budi' },
      afterData: { nama: 'Budi Santoso' },
      hashEntry: 'hash-entry-abc',
      createdAt: '2026-08-19T10:00:00Z',
    };
    mockAdminService.getAuditLogDetail.mockReturnValue(of(mockDetail));

    component.openDetail(10);

    expect(mockAdminService.getAuditLogDetail).toHaveBeenCalledWith(10);
    expect(component.selectedLogDetail()).toEqual(mockDetail);

    component.closeDetail();
    expect(component.selectedLogDetail()).toBeNull();
  });
});
