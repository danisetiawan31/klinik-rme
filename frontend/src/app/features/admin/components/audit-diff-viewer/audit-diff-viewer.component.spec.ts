import { ComponentFixture, TestBed } from '@angular/core/testing';
import { AuditLogDetail } from '../../admin.types';
import { AuditDiffViewerComponent } from './audit-diff-viewer.component';

describe('AuditDiffViewerComponent', () => {
  let component: AuditDiffViewerComponent;
  let fixture: ComponentFixture<AuditDiffViewerComponent>;

  const mockLog: AuditLogDetail = {
    id: 15,
    tabelTarget: 'pasien',
    recordId: 4,
    actorUserId: 2,
    aksi: 'update',
    beforeData: {
      alamat: 'Jl. Melati No. 1',
      noTelp: '08123456789',
      nama: 'Budi Santoso',
    },
    afterData: {
      alamat: 'Jl. Mawar No. 10',
      noTelp: '08123456789',
      nama: 'Budi Santoso',
    },
    hashEntry: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    createdAt: '2026-08-19T08:30:00Z',
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AuditDiffViewerComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(AuditDiffViewerComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('log', mockLog);
    fixture.detectChanges();
  });

  it('should create and compute diff entries properly', () => {
    expect(component).toBeTruthy();
    const diffs = component.diffEntries();
    expect(diffs.length).toBe(3);

    const alamatDiff = diffs.find((d) => d.key === 'alamat');
    expect(alamatDiff?.isModified).toBe(true);
    expect(alamatDiff?.beforeVal).toBe('Jl. Melati No. 1');
    expect(alamatDiff?.afterVal).toBe('Jl. Mawar No. 10');

    const noTelpDiff = diffs.find((d) => d.key === 'noTelp');
    expect(noTelpDiff?.isModified).toBe(false);
  });

  it('should detect rekam_medis clinical target and show warning', () => {
    expect(component.isRekamMedis()).toBe(false);

    const medicalLog: AuditLogDetail = {
      ...mockLog,
      tabelTarget: 'rekam_medis',
    };
    fixture.componentRef.setInput('log', medicalLog);
    fixture.detectChanges();

    expect(component.isRekamMedis()).toBe(true);
  });

  it('should emit closed event on onClose', () => {
    const emitSpy = vi.fn();
    component.closed.subscribe(emitSpy);

    component.onClose();
    expect(emitSpy).toHaveBeenCalled();
  });
});
