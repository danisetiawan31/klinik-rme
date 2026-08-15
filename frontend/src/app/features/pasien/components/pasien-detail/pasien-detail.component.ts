import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toast } from '@spartan-ng/brain/sonner';
import { AuthService } from '../../../../core/auth/auth.service';
import { KlinikService } from '../../../../core/klinik/klinik.service';
import { SensitiveValueComponent } from '../../../../shared/components/sensitive-value/sensitive-value.component';
import { AntrianService } from '../../../antrian/antrian.service';
import { PasienService } from '../../pasien.service';
import { Pasien } from '../../pasien.types';

@Component({
  selector: 'app-pasien-detail',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [RouterLink, SensitiveValueComponent],
  templateUrl: './pasien-detail.component.html',
})
export class PasienDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private pasienService = inject(PasienService);
  private antrianService = inject(AntrianService);
  private klinikService = inject(KlinikService);
  private authService = inject(AuthService);

  readonly pasien = signal<Pasien | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);
  readonly successMessage = signal<string | null>(null);

  // RBAC permissions: Petugas and Admin can manage patient & register to queue
  readonly isStaff = computed(() => {
    const roles = this.authService.currentUser()?.roles || [];
    return roles.includes('petugas') || roles.includes('admin');
  });
  readonly canEdit = this.isStaff;

  // Clinic operating hours state
  readonly klinikInfo = this.klinikService.klinikInfo;
  readonly isKlinikBuka = computed(() => this.klinikService.isKlinikBuka(this.klinikInfo()));

  // Queue registration dialog state (Tahap 3)
  readonly showDaftarModal = signal<boolean>(false);
  readonly isPriority = signal<boolean>(false);
  readonly priorityReason = signal<string>('');
  readonly formError = signal<string | null>(null);
  readonly isSubmitting = signal<boolean>(false);

  ngOnInit(): void {
    const nav = this.router.getCurrentNavigation();
    const state = (nav?.extras?.state ?? history.state) as { successMessage?: string };
    if (state?.successMessage) {
      this.successMessage.set(state.successMessage);
      toast.success(state.successMessage);
    }

    if (!this.klinikInfo()) {
      this.klinikService.fetchKlinikInfo().subscribe();
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
          err?.error?.message ??
          'Gagal mengambil detail pasien. Silakan coba lagi.';
        this.errorMessage.set(msg);
        toast.error(msg);
      },
    });
  }

  /**
   * Buka modal pendaftaran antrian & reset state form
   */
  openDaftarModal(): void {
    this.isPriority.set(false);
    this.priorityReason.set('');
    this.formError.set(null);
    this.showDaftarModal.set(true);
  }

  /**
   * Tutup modal pendaftaran antrian
   */
  closeDaftarModal(): void {
    this.showDaftarModal.set(false);
    this.formError.set(null);
  }

  /**
   * Submit pendaftaran pasien ke antrian hari ini (POST /api/v1/kunjungan)
   */
  submitDaftarAntrian(): void {
    const p = this.pasien();
    if (!p || this.isSubmitting()) return;

    // Client-side strict validation: priorityReason is required if isPriority is true
    if (this.isPriority() && !this.priorityReason().trim()) {
      this.formError.set('Alasan prioritas wajib diisi jika status prioritas diaktifkan.');
      return;
    }

    this.formError.set(null);
    this.isSubmitting.set(true);

    this.antrianService
      .create({
        pasienId: p.id,
        isPriority: this.isPriority(),
        priorityReason: this.isPriority() ? this.priorityReason().trim() : undefined,
      })
      .subscribe({
        next: (res) => {
          this.isSubmitting.set(false);
          this.showDaftarModal.set(false);
          const msg = `Pasien berhasil didaftarkan ke antrian hari ini dengan Nomor Antrian #${res.nomorAntrian}.`;
          this.successMessage.set(msg);
          toast.success(msg);
          // Refetch patient detail to update riwayatKunjunganRingkas without page redirect
          this.fetchDetail(p.id);
        },
        error: (err: any) => {
          this.isSubmitting.set(false);
          // Modal remains OPEN on failure so staff sees the error and can retry or cancel
          let msg = 'Gagal mendaftarkan pasien ke antrian.';
          if (err?.error?.code === 'KLINIK_TUTUP') {
            msg = 'Pendaftaran antrian sudah ditutup untuk hari ini.';
          } else if (err?.error?.message) {
            msg = err.error.message;
          } else if (err?.error?.error?.message) {
            msg = err.error.error.message;
          }
          this.errorMessage.set(msg);
          toast.error(msg);
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
