import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { environment } from '../../../environments/environment';
import { AdminService } from './admin.service';
import {
  AdminUser,
  AuditLogDetail,
  AuditLogSummary,
  CreateAdminUserRequest,
  CreateAdminUserResponse,
} from './admin.types';

describe('AdminService', () => {
  let service: AdminService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), AdminService],
    });
    service = TestBed.inject(AdminService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should fetch users list with pagination and X-Total-Count header', () => {
    const mockUsers: AdminUser[] = [
      { id: 1, nama: 'Admin User', email: 'admin@klinik.com', roles: ['admin'] },
      { id: 2, nama: 'dr. Sarah', email: 'sarah@klinik.com', roles: ['dokter'] },
    ];

    service.getUsers(2, 5).subscribe((res) => {
      expect(res.users).toEqual(mockUsers);
      expect(res.totalCount).toBe(12);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/admin/users?page=2&limit=5`);
    expect(req.request.method).toBe('GET');
    req.flush(mockUsers, {
      headers: { 'X-Total-Count': '12' },
    });
  });

  it('should create user via POST /admin/users', () => {
    const payload: CreateAdminUserRequest = {
      nama: 'Petugas Baru',
      email: 'petugas@klinik.com',
      roles: ['petugas'],
    };
    const mockResponse: CreateAdminUserResponse = {
      id: 3,
      nama: 'Petugas Baru',
      email: 'petugas@klinik.com',
      roles: ['petugas'],
      inviteLink: 'http://localhost:4200/set-password?token=invite-token-123',
    };

    service.createUser(payload).subscribe((res) => {
      expect(res).toEqual(mockResponse);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/admin/users`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(payload);
    req.flush(mockResponse, { status: 201, statusText: 'Created' });
  });

  it('should resend invite via POST /admin/users/:id/resend-invite', () => {
    service.resendInvite(5).subscribe();

    const req = httpTesting.expectOne(`${environment.apiUrl}/admin/users/5/resend-invite`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({});
    req.flush(null, { status: 204, statusText: 'No Content' });
  });

  it('should update user biodata via PATCH /admin/users/:id', () => {
    const payload = { nama: 'dr. Sarah Updated' };
    const mockUser: AdminUser = {
      id: 2,
      nama: 'dr. Sarah Updated',
      email: 'sarah@klinik.com',
      roles: ['dokter'],
    };

    service.updateUser(2, payload).subscribe((res) => {
      expect(res).toEqual(mockUser);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/admin/users/2`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual(payload);
    req.flush(mockUser);
  });

  it('should update user roles via PATCH /admin/users/:id/roles', () => {
    const payload = { roles: ['petugas', 'admin'] };
    const mockUser: AdminUser = {
      id: 1,
      nama: 'Admin User',
      email: 'admin@klinik.com',
      roles: ['petugas', 'admin'],
    };

    service.updateUserRoles(1, payload).subscribe((res) => {
      expect(res).toEqual(mockUser);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/admin/users/1/roles`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual(payload);
    req.flush(mockUser);
  });

  it('should fetch filtered audit logs with query params and X-Total-Count', () => {
    const mockLogs: AuditLogSummary[] = [
      {
        id: 10,
        tabelTarget: 'pasien',
        recordId: 5,
        actorUserId: 1,
        aksi: 'update',
        createdAt: '2026-08-19T10:00:00Z',
      },
    ];

    service
      .getAuditLogs({ tabelTarget: 'pasien', recordId: 5, actorId: 1, page: 1, limit: 10 })
      .subscribe((res) => {
        expect(res.logs).toEqual(mockLogs);
        expect(res.totalCount).toBe(1);
      });

    const req = httpTesting.expectOne(
      `${environment.apiUrl}/admin/audit-log?tabelTarget=pasien&recordId=5&actorId=1&page=1&limit=10`
    );
    expect(req.request.method).toBe('GET');
    req.flush(mockLogs, {
      headers: { 'X-Total-Count': '1' },
    });
  });

  it('should fetch audit log detail via GET /admin/audit-log/:id', () => {
    const mockDetail: AuditLogDetail = {
      id: 10,
      tabelTarget: 'pasien',
      recordId: 5,
      actorUserId: 1,
      aksi: 'update',
      beforeData: { nama: 'Budi' },
      afterData: { nama: 'Budi Santoso' },
      hashEntry: 'sha256-hash-abc',
      createdAt: '2026-08-19T10:00:00Z',
    };

    service.getAuditLogDetail(10).subscribe((res) => {
      expect(res).toEqual(mockDetail);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/admin/audit-log/10`);
    expect(req.request.method).toBe('GET');
    req.flush(mockDetail);
  });

  it('should regenerate display token via POST /admin/klinik/:id/display-token/regenerate', () => {
    const mockResponse = { displayToken: 'new-raw-display-token-secret' };

    service.regenerateDisplayToken(1).subscribe((res) => {
      expect(res.displayToken).toBe('new-raw-display-token-secret');
    });

    const req = httpTesting.expectOne(
      `${environment.apiUrl}/admin/klinik/1/display-token/regenerate`
    );
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({});
    req.flush(mockResponse);
  });
});
