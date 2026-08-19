import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import {
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideAlertCircle,
  lucideAlertTriangle,
  lucideCheck,
  lucideCheckCircle2,
  lucideEdit3,
  lucideMail,
  lucidePlus,
  lucideSend,
  lucideShield,
  lucideShieldAlert,
  lucideUserCheck,
  lucideUserPlus,
  lucideUsers,
} from '@ng-icons/lucide';
import { toast } from '@spartan-ng/brain/sonner';
import { PaginationComponent } from '../../../../shared/components/pagination/pagination.component';
import { RevealOnceSecretComponent } from '../../../../shared/components/reveal-once-secret/reveal-once-secret.component';
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
import { AdminUser, CreateAdminUserRequest } from '../../admin.types';

@Component({
  selector: 'app-admin-users',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    PaginationComponent,
    RevealOnceSecretComponent,
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
      lucideUsers,
      lucideUserPlus,
      lucideEdit3,
      lucideShield,
      lucideShieldAlert,
      lucideSend,
      lucideCheckCircle2,
      lucideAlertCircle,
      lucideAlertTriangle,
      lucidePlus,
      lucideMail,
      lucideUserCheck,
      lucideCheck,
    }),
  ],
  templateUrl: './admin-users.component.html',
})
export class AdminUsersComponent implements OnInit {
  private adminService = inject(AdminService);
  private fb = inject(FormBuilder);

  // Table & pagination state
  readonly users = signal<AdminUser[]>([]);
  readonly totalCount = signal<number>(0);
  readonly currentPage = signal<number>(1);
  readonly limit = 10;
  readonly isLoading = signal<boolean>(true);
  readonly errorMessage = signal<string | null>(null);

  // Invite Modal state
  readonly showInviteModal = signal<boolean>(false);
  readonly isSubmittingInvite = signal<boolean>(false);
  readonly inviteForm: FormGroup = this.fb.group({
    nama: ['', [Validators.required, Validators.minLength(2)]],
    email: ['', [Validators.required, Validators.email]],
    rolePetugas: [false],
    roleDokter: [false],
    roleAdmin: [false],
  });

  // Reveal Invite Link Modal state
  readonly revealInviteLink = signal<string | null>(null);

  // Edit User Modal state
  readonly editingUser = signal<AdminUser | null>(null);
  readonly isSubmittingEdit = signal<boolean>(false);
  readonly editForm: FormGroup = this.fb.group({
    nama: ['', [Validators.required, Validators.minLength(2)]],
    email: ['', [Validators.required, Validators.email]],
  });

  // Edit Roles Modal state
  readonly editingRolesUser = signal<AdminUser | null>(null);
  readonly isSubmittingRoles = signal<boolean>(false);
  readonly rolesForm: FormGroup = this.fb.group({
    rolePetugas: [false],
    roleDokter: [false],
    roleAdmin: [false],
  });

  // Action loading state
  readonly resendingUserId = signal<number | null>(null);

  ngOnInit(): void {
    this.loadUsers(1);
  }

  loadUsers(page: number): void {
    this.isLoading.set(true);
    this.errorMessage.set(null);
    this.currentPage.set(page);

    this.adminService.getUsers(page, this.limit).subscribe({
      next: (res) => {
        this.users.set(res.users);
        this.totalCount.set(res.totalCount);
        this.isLoading.set(false);
      },
      error: (err) => {
        this.isLoading.set(false);
        this.errorMessage.set(err?.error?.message || 'Gagal memuat daftar pengguna.');
      },
    });
  }

  // --- Invite Modal Methods ---

  openInviteModal(): void {
    this.inviteForm.reset({
      nama: '',
      email: '',
      rolePetugas: true,
      roleDokter: false,
      roleAdmin: false,
    });
    this.showInviteModal.set(true);
  }

  closeInviteModal(): void {
    this.showInviteModal.set(false);
  }

  isInviteRolesConflict(): boolean {
    const isDokter = this.inviteForm.get('roleDokter')?.value;
    const isAdmin = this.inviteForm.get('roleAdmin')?.value;
    return Boolean(isDokter && isAdmin);
  }

  hasAtLeastOneInviteRole(): boolean {
    const isPetugas = this.inviteForm.get('rolePetugas')?.value;
    const isDokter = this.inviteForm.get('roleDokter')?.value;
    const isAdmin = this.inviteForm.get('roleAdmin')?.value;
    return Boolean(isPetugas || isDokter || isAdmin);
  }

  submitInvite(): void {
    if (this.inviteForm.invalid || this.isInviteRolesConflict() || !this.hasAtLeastOneInviteRole()) {
      return;
    }

    const val = this.inviteForm.value;
    const roles: string[] = [];
    if (val.rolePetugas) roles.push('petugas');
    if (val.roleDokter) roles.push('dokter');
    if (val.roleAdmin) roles.push('admin');

    const payload: CreateAdminUserRequest = {
      nama: val.nama.trim(),
      email: val.email.trim().toLowerCase(),
      roles,
    };

    this.isSubmittingInvite.set(true);

    this.adminService.createUser(payload).subscribe({
      next: (res) => {
        this.isSubmittingInvite.set(false);
        this.closeInviteModal();
        toast.success(`Pengguna ${res.nama} berhasil diundang!`);
        this.loadUsers(this.currentPage());

        if (res.inviteLink) {
          this.revealInviteLink.set(res.inviteLink);
        }
      },
      error: (err) => {
        this.isSubmittingInvite.set(false);
        toast.error(err?.error?.message || 'Gagal mengundang pengguna.');
      },
    });
  }

  // --- Edit User Modal Methods ---

  openEditModal(user: AdminUser): void {
    this.editingUser.set(user);
    this.editForm.patchValue({
      nama: user.nama,
      email: user.email,
    });
  }

  closeEditModal(): void {
    this.editingUser.set(null);
  }

  submitEdit(): void {
    const user = this.editingUser();
    if (!user || this.editForm.invalid) return;

    const val = this.editForm.value;
    this.isSubmittingEdit.set(true);

    this.adminService
      .updateUser(user.id, {
        nama: val.nama.trim(),
        email: val.email.trim().toLowerCase(),
      })
      .subscribe({
        next: (updated) => {
          this.isSubmittingEdit.set(false);
          this.closeEditModal();
          toast.success(`Data pengguna ${updated.nama} berhasil diperbarui.`);
          this.loadUsers(this.currentPage());
        },
        error: (err) => {
          this.isSubmittingEdit.set(false);
          toast.error(err?.error?.message || 'Gagal memperbarui pengguna.');
        },
      });
  }

  // --- Edit Roles Modal Methods ---

  openRolesModal(user: AdminUser): void {
    this.editingRolesUser.set(user);
    this.rolesForm.patchValue({
      rolePetugas: user.roles.includes('petugas'),
      roleDokter: user.roles.includes('dokter'),
      roleAdmin: user.roles.includes('admin'),
    });
  }

  closeRolesModal(): void {
    this.editingRolesUser.set(null);
  }

  isRolesConflict(): boolean {
    const isDokter = this.rolesForm.get('roleDokter')?.value;
    const isAdmin = this.rolesForm.get('roleAdmin')?.value;
    return Boolean(isDokter && isAdmin);
  }

  hasAtLeastOneRole(): boolean {
    const isPetugas = this.rolesForm.get('rolePetugas')?.value;
    const isDokter = this.rolesForm.get('roleDokter')?.value;
    const isAdmin = this.rolesForm.get('roleAdmin')?.value;
    return Boolean(isPetugas || isDokter || isAdmin);
  }

  submitRoles(): void {
    const user = this.editingRolesUser();
    if (!user || this.isRolesConflict() || !this.hasAtLeastOneRole()) return;

    const val = this.rolesForm.value;
    const roles: string[] = [];
    if (val.rolePetugas) roles.push('petugas');
    if (val.roleDokter) roles.push('dokter');
    if (val.roleAdmin) roles.push('admin');

    this.isSubmittingRoles.set(true);

    this.adminService.updateUserRoles(user.id, { roles }).subscribe({
      next: (updated) => {
        this.isSubmittingRoles.set(false);
        this.closeRolesModal();
        toast.success(`Peran pengguna ${updated.nama} berhasil diperbarui.`);
        this.loadUsers(this.currentPage());
      },
      error: (err) => {
        this.isSubmittingRoles.set(false);
        toast.error(err?.error?.message || 'Gagal memperbarui peran pengguna.');
      },
    });
  }

  // --- Resend Invite Method ---

  resendInvite(user: AdminUser): void {
    this.resendingUserId.set(user.id);

    this.adminService.resendInvite(user.id).subscribe({
      next: () => {
        this.resendingUserId.set(null);
        toast.success(`Tautan undangan berhasil dikirim ulang ke ${user.email}.`);
      },
      error: (err) => {
        this.resendingUserId.set(null);
        toast.error(err?.error?.message || 'Gagal mengirim ulang undangan.');
      },
    });
  }
}
