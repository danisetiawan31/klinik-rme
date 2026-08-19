import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { AuthService } from '../../core/auth/auth.service';
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
    };

    await TestBed.configureTestingModule({
      imports: [AdminDashboardComponent],
      providers: [
        { provide: AuthService, useValue: mockAuthService },
        { provide: AdminService, useValue: mockAdminService },
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
