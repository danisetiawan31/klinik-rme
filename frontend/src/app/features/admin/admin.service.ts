import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import {
  AdminUser,
  AdminUserListResult,
  AuditLogDetail,
  AuditLogFilterParams,
  AuditLogListResult,
  AuditLogSummary,
  CreateAdminUserRequest,
  CreateAdminUserResponse,
  RegenerateDisplayTokenResponse,
  UpdateAdminUserRequest,
  UpdateUserRolesRequest,
} from './admin.types';

@Injectable({
  providedIn: 'root',
})
export class AdminService {
  private http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/admin`;

  /**
   * Fetch paginated list of staff users (GET /api/v1/admin/users)
   * Reads header X-Total-Count for pagination state.
   */
  getUsers(page: number = 1, limit: number = 10): Observable<AdminUserListResult> {
    const params = new HttpParams()
      .set('page', page.toString())
      .set('limit', limit.toString());

    return this.http
      .get<AdminUser[]>(`${this.baseUrl}/users`, {
        params,
        observe: 'response',
      })
      .pipe(
        map((response) => {
          const totalHeader = response.headers.get('X-Total-Count');
          const totalCount = totalHeader ? parseInt(totalHeader, 10) : 0;
          return {
            users: response.body || [],
            totalCount: isNaN(totalCount) ? 0 : totalCount,
          };
        })
      );
  }

  /**
   * Create a new user and generate invite token (POST /api/v1/admin/users)
   */
  createUser(payload: CreateAdminUserRequest): Observable<CreateAdminUserResponse> {
    return this.http.post<CreateAdminUserResponse>(`${this.baseUrl}/users`, payload);
  }

  /**
   * Resend invite email with new token (POST /api/v1/admin/users/:id/resend-invite)
   */
  resendInvite(userId: number): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/users/${userId}/resend-invite`, {});
  }

  /**
   * Update user biodata (nama, email) (PATCH /api/v1/admin/users/:id)
   */
  updateUser(userId: number, payload: UpdateAdminUserRequest): Observable<AdminUser> {
    return this.http.patch<AdminUser>(`${this.baseUrl}/users/${userId}`, payload);
  }

  /**
   * Update user roles with mutual exclusivity enforcement (PATCH /api/v1/admin/users/:id/roles)
   */
  updateUserRoles(userId: number, payload: UpdateUserRolesRequest): Observable<AdminUser> {
    return this.http.patch<AdminUser>(`${this.baseUrl}/users/${userId}/roles`, payload);
  }

  /**
   * Fetch filtered and paginated audit logs (GET /api/v1/admin/audit-log)
   * Reads header X-Total-Count for pagination state.
   */
  getAuditLogs(filterParams: AuditLogFilterParams = {}): Observable<AuditLogListResult> {
    let params = new HttpParams();

    if (filterParams.tabelTarget?.trim()) {
      params = params.set('tabelTarget', filterParams.tabelTarget.trim());
    }
    if (filterParams.recordId !== undefined && filterParams.recordId > 0) {
      params = params.set('recordId', filterParams.recordId.toString());
    }
    if (filterParams.actorId !== undefined && filterParams.actorId > 0) {
      params = params.set('actorId', filterParams.actorId.toString());
    }

    const page = filterParams.page && filterParams.page > 0 ? filterParams.page : 1;
    const limit = filterParams.limit && filterParams.limit > 0 ? filterParams.limit : 10;

    params = params.set('page', page.toString()).set('limit', limit.toString());

    return this.http
      .get<AuditLogSummary[]>(`${this.baseUrl}/audit-log`, {
        params,
        observe: 'response',
      })
      .pipe(
        map((response) => {
          const totalHeader = response.headers.get('X-Total-Count');
          const totalCount = totalHeader ? parseInt(totalHeader, 10) : 0;
          return {
            logs: response.body || [],
            totalCount: isNaN(totalCount) ? 0 : totalCount,
          };
        })
      );
  }

  /**
   * Fetch detailed audit log with before/after diff and hashEntry (GET /api/v1/admin/audit-log/:id)
   */
  getAuditLogDetail(id: number): Observable<AuditLogDetail> {
    return this.http.get<AuditLogDetail>(`${this.baseUrl}/audit-log/${id}`);
  }

  /**
   * Regenerate display token for queue board (POST /api/v1/admin/klinik/:id/display-token/regenerate)
   */
  regenerateDisplayToken(
    klinikId: number = environment.defaultKlinikId
  ): Observable<RegenerateDisplayTokenResponse> {
    return this.http.post<RegenerateDisplayTokenResponse>(
      `${this.baseUrl}/klinik/${klinikId}/display-token/regenerate`,
      {}
    );
  }
}
