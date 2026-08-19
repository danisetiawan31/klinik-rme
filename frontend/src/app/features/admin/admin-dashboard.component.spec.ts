import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AuthService } from '../../core/auth/auth.service';
import { KlinikService } from '../../core/klinik/klinik.service';
import { AdminDashboardComponent } from './admin-dashboard.component';
import { AdminService } from './admin.service';

describe('AdminDashboardComponent', () => {
  let component: AdminDashboardComponent;
  let fixture: ComponentFixture<AdminDashboardComponent>;
  let mockAuthService: {
    currentUser: ReturnType<typeof vi.fn>;
  };
  let mockAdminService: {
    getUsers: ReturnType<typeof vi.fn>;
    getAuditLogs: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    mockAuthService = {
      currentUser: vi.fn().mockReturnValue({
        id: 1,
        nama: 'Super Administrator',
        roles: ['admin'],
      }),
    };

    mockAdminService = {
      getUsers: vi.fn().mockReturnValue(of({ users: [], totalCount: 0 })),
      getAuditLogs: vi.fn().mockReturnValue(of({ logs: [], totalCount: 0 })),
    };

    const mockKlinikService = {
      fetchKlinikInfo: vi.fn().mockReturnValue(of(null)),
      klinikInfo: vi.fn().mockReturnValue(null),
      isKlinikBuka: vi.fn().mockReturnValue(true),
    };

    await TestBed.configureTestingModule({
      imports: [AdminDashboardComponent],
      providers: [
        provideRouter([
          { path: 'admin', component: AdminDashboardComponent },
          { path: 'admin/:subtab', component: AdminDashboardComponent },
        ]),
        { provide: AuthService, useValue: mockAuthService },
        { provide: AdminService, useValue: mockAdminService },
        { provide: KlinikService, useValue: mockKlinikService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AdminDashboardComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and default to users tab', () => {
    expect(component).toBeTruthy();
    expect(component.activeTab()).toBe('users');
  });

  it('should switch tabs when setTab is called', () => {
    component.setTab('audit-log');
    expect(component.activeTab()).toBe('audit-log');

    component.setTab('klinik');
    expect(component.activeTab()).toBe('klinik');
  });
});
