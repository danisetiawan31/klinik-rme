import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { AdminService } from '../../admin.service';
import { AdminUser, CreateAdminUserResponse } from '../../admin.types';
import { AdminUsersComponent } from './admin-users.component';

describe('AdminUsersComponent', () => {
  let component: AdminUsersComponent;
  let fixture: ComponentFixture<AdminUsersComponent>;
  let mockAdminService: {
    getUsers: ReturnType<typeof vi.fn>;
    createUser: ReturnType<typeof vi.fn>;
    updateUser: ReturnType<typeof vi.fn>;
    updateUserRoles: ReturnType<typeof vi.fn>;
    resendInvite: ReturnType<typeof vi.fn>;
  };

  const mockUsersList: AdminUser[] = [
    { id: 1, nama: 'Admin Super', email: 'admin@klinik.com', roles: ['admin'] },
    { id: 2, nama: 'dr. Sarah', email: 'sarah@klinik.com', roles: ['dokter'] },
  ];

  beforeEach(async () => {
    mockAdminService = {
      getUsers: vi.fn().mockReturnValue(of({ users: mockUsersList, totalCount: 2 })),
      createUser: vi.fn(),
      updateUser: vi.fn(),
      updateUserRoles: vi.fn(),
      resendInvite: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [AdminUsersComponent],
      providers: [{ provide: AdminService, useValue: mockAdminService }],
    }).compileComponents();

    fixture = TestBed.createComponent(AdminUsersComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and load users list on init', () => {
    expect(component).toBeTruthy();
    expect(mockAdminService.getUsers).toHaveBeenCalledWith(1, 10);
    expect(component.users()).toEqual(mockUsersList);
    expect(component.totalCount()).toBe(2);
    expect(component.isLoading()).toBe(false);
  });

  it('should open invite modal and submit invite with reveal link', () => {
    const mockCreateResponse: CreateAdminUserResponse = {
      id: 3,
      nama: 'Petugas Baru',
      email: 'petugas@klinik.com',
      roles: ['petugas'],
      inviteLink: 'http://localhost:4200/set-password?token=token-123',
    };
    mockAdminService.createUser.mockReturnValue(of(mockCreateResponse));

    component.openInviteModal();
    expect(component.showInviteModal()).toBe(true);

    component.inviteForm.patchValue({
      nama: 'Petugas Baru',
      email: 'petugas@klinik.com',
      rolePetugas: true,
      roleDokter: false,
      roleAdmin: false,
    });

    component.submitInvite();

    expect(mockAdminService.createUser).toHaveBeenCalledWith({
      nama: 'Petugas Baru',
      email: 'petugas@klinik.com',
      roles: ['petugas'],
    });
    expect(component.showInviteModal()).toBe(false);
    expect(component.revealInviteLink()).toBe('http://localhost:4200/set-password?token=token-123');
  });

  it('should prevent invite submission if both dokter and admin roles are selected', () => {
    component.openInviteModal();
    component.inviteForm.patchValue({
      nama: 'Conflicted User',
      email: 'conflict@klinik.com',
      rolePetugas: false,
      roleDokter: true,
      roleAdmin: true,
    });

    expect(component.isInviteRolesConflict()).toBe(true);

    component.submitInvite();
    expect(mockAdminService.createUser).not.toHaveBeenCalled();
  });

  it('should open edit modal and submit user biodata update', () => {
    const updatedUser: AdminUser = {
      id: 2,
      nama: 'dr. Sarah Updated',
      email: 'sarah.new@klinik.com',
      roles: ['dokter'],
    };
    mockAdminService.updateUser.mockReturnValue(of(updatedUser));

    component.openEditModal(mockUsersList[1]);
    expect(component.editingUser()).toEqual(mockUsersList[1]);
    expect(component.editForm.value.nama).toBe('dr. Sarah');

    component.editForm.patchValue({
      nama: 'dr. Sarah Updated',
      email: 'sarah.new@klinik.com',
    });

    component.submitEdit();

    expect(mockAdminService.updateUser).toHaveBeenCalledWith(2, {
      nama: 'dr. Sarah Updated',
      email: 'sarah.new@klinik.com',
    });
    expect(component.editingUser()).toBeNull();
  });

  it('should open roles modal, validate mutual exclusivity and submit roles update', () => {
    const updatedUser: AdminUser = {
      id: 1,
      nama: 'Admin Super',
      email: 'admin@klinik.com',
      roles: ['petugas', 'admin'],
    };
    mockAdminService.updateUserRoles.mockReturnValue(of(updatedUser));

    component.openRolesModal(mockUsersList[0]);
    expect(component.editingRolesUser()).toEqual(mockUsersList[0]);

    // Test conflict detection
    component.rolesForm.patchValue({
      rolePetugas: false,
      roleDokter: true,
      roleAdmin: true,
    });
    expect(component.isRolesConflict()).toBe(true);

    component.submitRoles();
    expect(mockAdminService.updateUserRoles).not.toHaveBeenCalled();

    // Valid role selection
    component.rolesForm.patchValue({
      rolePetugas: true,
      roleDokter: false,
      roleAdmin: true,
    });
    expect(component.isRolesConflict()).toBe(false);

    component.submitRoles();

    expect(mockAdminService.updateUserRoles).toHaveBeenCalledWith(1, {
      roles: ['petugas', 'admin'],
    });
    expect(component.editingRolesUser()).toBeNull();
  });

  it('should call resendInvite and update loading state', () => {
    mockAdminService.resendInvite.mockReturnValue(of(undefined));

    component.resendInvite(mockUsersList[1]);

    expect(mockAdminService.resendInvite).toHaveBeenCalledWith(2);
    expect(component.resendingUserId()).toBeNull();
  });
});
