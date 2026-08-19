/**
 * Types & Data Transfer Objects for Admin Feature
 * Aligned with docs/api-contract.md and AGENTS.md conventions.
 */

export interface AdminUser {
  id: number;
  nama: string;
  email: string;
  roles: string[];
}

export interface AdminUserListResult {
  users: AdminUser[];
  totalCount: number;
}

export interface CreateAdminUserRequest {
  nama: string;
  email: string;
  roles: string[];
}

export interface CreateAdminUserResponse {
  id: number;
  nama: string;
  email: string;
  roles: string[];
  inviteLink: string;
}

export interface UpdateAdminUserRequest {
  nama?: string;
  email?: string;
}

export interface UpdateUserRolesRequest {
  roles: string[];
}

export interface AuditLogSummary {
  id: number;
  tabelTarget: string;
  recordId: number;
  actorUserId: number;
  aksi: string;
  createdAt: string;
}

export interface AuditLogListResult {
  logs: AuditLogSummary[];
  totalCount: number;
}

export interface AuditLogDetail {
  id: number;
  tabelTarget: string;
  recordId: number;
  actorUserId: number;
  aksi: string;
  beforeData: Record<string, any> | null;
  afterData: Record<string, any> | null;
  hashEntry: string;
  createdAt: string;
}

export interface AuditLogFilterParams {
  tabelTarget?: string;
  recordId?: number;
  actorId?: number;
  page?: number;
  limit?: number;
}

export interface RegenerateDisplayTokenResponse {
  displayToken: string;
}
