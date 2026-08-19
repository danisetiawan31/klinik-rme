import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { KlinikService } from '../../../../core/klinik/klinik.service';
import { AdminService } from '../../admin.service';
import { AdminKlinikComponent } from './admin-klinik.component';

describe('AdminKlinikComponent', () => {
  let component: AdminKlinikComponent;
  let fixture: ComponentFixture<AdminKlinikComponent>;
  let mockAdminService: {
    regenerateDisplayToken: ReturnType<typeof vi.fn>;
  };
  let mockKlinikService: {
    fetchKlinikInfo: ReturnType<typeof vi.fn>;
    klinikInfo: ReturnType<typeof vi.fn>;
    isKlinikBuka: ReturnType<typeof vi.fn>;
  };

  const mockKlinikData = {
    id: 1,
    nama: 'Klinik RME Sehat',
    jamBuka: '08:00',
    jamTutup: '21:00',
    isBuka: true,
  };

  beforeEach(async () => {
    mockAdminService = {
      regenerateDisplayToken: vi.fn(),
    };

    mockKlinikService = {
      fetchKlinikInfo: vi.fn().mockReturnValue(of(mockKlinikData)),
      klinikInfo: vi.fn().mockReturnValue(mockKlinikData),
      isKlinikBuka: vi.fn().mockReturnValue(true),
    };

    await TestBed.configureTestingModule({
      imports: [AdminKlinikComponent],
      providers: [
        { provide: AdminService, useValue: mockAdminService },
        { provide: KlinikService, useValue: mockKlinikService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AdminKlinikComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and fetch clinic info on init', () => {
    expect(component).toBeTruthy();
    expect(mockKlinikService.fetchKlinikInfo).toHaveBeenCalled();
  });

  it('should open and close confirm modal properly', () => {
    expect(component.showConfirmModal()).toBe(false);

    component.openConfirmModal();
    expect(component.showConfirmModal()).toBe(true);

    component.closeConfirmModal();
    expect(component.showConfirmModal()).toBe(false);
  });

  it('should regenerate display token on confirmation and set newDisplayToken', () => {
    mockAdminService.regenerateDisplayToken.mockReturnValue(
      of({ displayToken: 'newly-generated-raw-secret' })
    );

    component.openConfirmModal();
    component.confirmRegenerate();

    expect(mockAdminService.regenerateDisplayToken).toHaveBeenCalledWith(1);
    expect(component.isRegenerating()).toBe(false);
    expect(component.showConfirmModal()).toBe(false);
    expect(component.newDisplayToken()).toBe('newly-generated-raw-secret');
  });

  it('should handle error when regenerate fails', () => {
    mockAdminService.regenerateDisplayToken.mockReturnValue(
      throwError(() => ({ error: { message: 'Server error' } }))
    );

    component.openConfirmModal();
    component.confirmRegenerate();

    expect(component.isRegenerating()).toBe(false);
    expect(component.newDisplayToken()).toBeNull();
  });
});
