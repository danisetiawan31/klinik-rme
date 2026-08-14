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
  template: `
    @if (errorMessage()) {
      <app-toast
        [message]="errorMessage() || ''"
        type="error"
        (dismiss)="errorMessage.set(null)"
      />
    }
    @if (successMessage()) {
      <app-toast
        [message]="successMessage() || ''"
        type="success"
        (dismiss)="successMessage.set(null)"
      />
    }

    <!-- ── Page Wrapper (Zona Content — DESIGN.md §1.1) ── -->
    <div class="min-h-full p-6 bg-background">
      <!-- Top nav back link -->
      <div class="mb-4">
        <a
          routerLink="/pasien"
          class="font-sans text-sm font-semibold text-primary no-underline inline-flex items-center gap-1"
        >
          &larr; Kembali ke Daftar Pasien
        </a>
      </div>

      @if (isLoading()) {
        <div class="p-12 text-center text-muted-foreground font-sans">
          <svg
            class="kl-spinner text-primary mx-auto mb-3 block"
            xmlns="http://www.w3.org/2000/svg"
            width="24" height="24" viewBox="0 0 24 24"
            fill="none" stroke="currentColor" stroke-width="2.5"
            stroke-linecap="round" aria-hidden="true"
          >
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          Memuat detail pasien...
        </div>
      } @else if (pasien()) {
        <!-- Header banner -->
        <div class="bg-card border border-border rounded-md shadow-2 p-6 mb-6 flex items-center justify-between flex-wrap gap-4">
          <div>
            <div class="flex items-center gap-3 mb-1">
              <h1 class="font-heading text-2xl font-bold text-foreground m-0">
                {{ pasien()!.nama }}
              </h1>
              <span class="font-sans text-xs px-2 py-0.5 rounded-full bg-muted text-muted-foreground font-semibold">
                ID Pasien #{{ pasien()!.id }}
              </span>
            </div>
            <p class="font-sans text-sm text-muted-foreground m-0">
              Versi Data: v{{ pasien()!.version }}
            </p>
          </div>

          @if (canEdit()) {
            <a
              [routerLink]="['/pasien', pasien()!.id, 'edit']"
              class="kl-btn-secondary no-underline inline-flex items-center gap-2"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="15" height="15" viewBox="0 0 24 24"
                fill="none" stroke="currentColor" stroke-width="2"
                stroke-linecap="round" stroke-linejoin="round"
                aria-hidden="true"
              >
                <path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
              </svg>
              Edit Biodata
            </a>
          }
        </div>

        <!-- Grid layout: Biodata + Riwayat Ringkas -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <!-- ── Card 1: Biodata Lengkap ── -->
          <div class="bg-card border border-border rounded-md shadow-2 p-6">
            <h2 class="font-heading text-lg font-bold text-foreground mb-4 pb-2 border-b border-border">
              Biodata Lengkap
            </h2>

            <dl class="grid grid-cols-[140px_1fr] gap-x-4 gap-y-3 font-sans text-sm m-0">
              <dt class="text-muted-foreground font-medium">NIK</dt>
              <dd class="m-0 font-semibold">
                @if (pasien()!.nik) {
                  <app-sensitive-value
                    mode="display"
                    [displayValue]="pasien()!.nik!"
                  />
                } @else {
                  <span class="text-muted-foreground italic">
                    (Tidak ada NIK)
                  </span>
                }
              </dd>

              <dt class="text-muted-foreground font-medium">Jenis Kelamin</dt>
              <dd class="m-0">
                {{ pasien()!.jenisKelamin === 'L' ? 'Laki-laki' : 'Perempuan' }}
              </dd>

              <dt class="text-muted-foreground font-medium">Tanggal Lahir</dt>
              <dd class="m-0">
                {{ formatDate(pasien()!.tanggalLahir) }}
              </dd>

              <dt class="text-muted-foreground font-medium">No. Telepon</dt>
              <dd class="m-0">
                {{ pasien()!.noTelp }}
              </dd>

              <dt class="text-muted-foreground font-medium">Alamat</dt>
              <dd class="m-0 whitespace-pre-wrap">
                {{ pasien()!.alamat }}
              </dd>

              <dt class="text-muted-foreground font-medium">Status Consent</dt>
              <dd class="m-0">
                @if (pasien()!.consent) {
                  <span class="text-accent font-semibold inline-flex items-center gap-1">
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                    Disetujui
                  </span>
                } @else {
                  <span class="text-destructive font-semibold">
                    Belum disetujui
                  </span>
                }
              </dd>
            </dl>
          </div>

          <!-- ── Card 2: Riwayat Kunjungan Ringkas ── -->
          <div class="bg-card border border-border rounded-md shadow-2 p-6">
            <h2 class="font-heading text-lg font-bold text-foreground mb-4 pb-2 border-b border-border">
              Riwayat Kunjungan Ringkas
            </h2>

            @if (pasien()!.riwayatKunjunganRingkas && pasien()!.riwayatKunjunganRingkas.length > 0) {
              <ul class="list-none p-0 m-0 flex flex-col gap-2">
                @for (kunjungan of pasien()!.riwayatKunjunganRingkas; track kunjungan.kunjunganId) {
                  <li class="flex items-center justify-between p-2.5 sm:px-3 bg-muted rounded-sm font-sans text-sm">
                    <div>
                      <span class="font-semibold text-foreground">
                        Kunjungan #{{ kunjungan.kunjunganId }}
                      </span>
                      <span class="block text-xs text-muted-foreground">
                        {{ formatDate(kunjungan.tanggal) }}
                      </span>
                    </div>

                    <!-- Status badge (DESIGN.md §2 status colors) -->
                    <span [class]="getStatusBadgeClass(kunjungan.status)">
                      {{ getStatusLabel(kunjungan.status) }}
                    </span>
                  </li>
                }
              </ul>
            } @else {
              <div class="p-8 text-center text-muted-foreground font-sans text-sm">
                Belum ada riwayat kunjungan recorded.
              </div>
            }
          </div>
        </div>
      }
    </div>
  `,
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
