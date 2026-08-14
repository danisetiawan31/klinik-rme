import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { AuthService } from '../../../../core/auth/auth.service';
import { SensitiveValueComponent } from '../../../../shared/components/sensitive-value/sensitive-value.component';
import { ToastComponent } from '../../../../shared/components/toast/toast.component';
import { PasienService } from '../../pasien.service';
import { Pasien } from '../../pasien.types';

@Component({
  selector: 'app-pasien-detail',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [RouterLink, SensitiveValueComponent, ToastComponent],
  templateUrl: './pasien-detail.component.html',
})
export class PasienDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private pasienService = inject(PasienService);
  private authService = inject(AuthService);

  readonly pasien = signal<Pasien | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);
  readonly successMessage = signal<string | null>(null);

  readonly canEdit = computed(() => {
    const roles = this.authService.currentUser()?.roles || [];
    return roles.includes('petugas') || roles.includes('admin');
  });

  ngOnInit(): void {
    const state = history.state as { successMessage?: string };
    if (state?.successMessage) {
      this.successMessage.set(state.successMessage);
    }

    const idParam = this.route.snapshot.paramMap.get('id');
    const id = idParam ? parseInt(idParam, 10) : 0;

    if (!id || isNaN(id)) {
      this.errorMessage.set('ID pasien tidak valid.');
      return;
    }

    this.fetchDetail(id);
  }

  fetchDetail(id: number): void {
    this.isLoading.set(true);
    this.errorMessage.set(null);

    this.pasienService.getById(id).subscribe({
      next: (data) => {
        this.isLoading.set(false);
        this.pasien.set(data);
      },
      error: (err: any) => {
        this.isLoading.set(false);
        const msg =
          err?.error?.error?.message ??
          'Gagal mengambil detail pasien. Silakan coba lagi.';
        this.errorMessage.set(msg);
      },
    });
  }

  formatDate(isoDateStr: string): string {
    if (!isoDateStr) return '-';
    try {
      const d = new Date(isoDateStr);
      return d.toLocaleDateString('id-ID', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
      });
    } catch {
      return isoDateStr;
    }
  }

  getStatusLabel(status: string): string {
    switch (status) {
      case 'menunggu': return 'Menunggu';
      case 'dipanggil': return 'Dipanggil';
      case 'selesai': return 'Selesai';
      case 'tidak_hadir': return 'Tidak Hadir';
      default: return status;
    }
  }

  getStatusBadgeClass(status: string): string {
    const base = 'text-xs font-semibold px-2 py-0.5 rounded-full';
    switch (status) {
      case 'menunggu':
        return `${base} bg-muted text-warning-foreground border border-warning`;
      case 'dipanggil':
        return `${base} bg-primary text-primary-foreground`;
      case 'selesai':
        return `${base} bg-accent text-accent-foreground`;
      case 'tidak_hadir':
        return `${base} bg-muted text-muted-foreground border border-border`;
      default:
        return `${base} bg-muted text-muted-foreground`;
    }
  }
}
