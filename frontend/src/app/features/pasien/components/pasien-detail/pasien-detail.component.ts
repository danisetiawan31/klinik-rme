import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
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

    <!-- ── Page Wrapper (Zona Content — DESIGN.md §1.1) ── -->
    <div
      style="
        min-height: 100%;
        padding: var(--space-6);
        background-color: var(--color-background);
      "
    >
      <!-- Top nav back link -->
      <div style="margin-bottom: var(--space-4);">
        <a
          routerLink="/pasien"
          style="
            font-family: var(--font-body);
            font-size: var(--text-sm);
            font-weight: 600;
            color: var(--color-primary);
            text-decoration: none;
            display: inline-flex;
            align-items: center;
            gap: var(--space-1);
          "
        >
          &larr; Kembali ke Daftar Pasien
        </a>
      </div>

      @if (isLoading()) {
        <div
          style="
            padding: var(--space-12);
            text-align: center;
            color: var(--color-muted-foreground);
            font-family: var(--font-body);
          "
        >
          <svg
            class="kl-spinner"
            xmlns="http://www.w3.org/2000/svg"
            width="24" height="24" viewBox="0 0 24 24"
            fill="none" stroke="var(--color-primary)" stroke-width="2.5"
            stroke-linecap="round" aria-hidden="true"
            style="margin: 0 auto 12px; display:block;"
          >
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          Memuat detail pasien...
        </div>
      } @else if (pasien()) {
        <!-- Header banner -->
        <div
          style="
            background: var(--color-card);
            border: 1px solid var(--color-border);
            border-radius: var(--radius-md);
            box-shadow: var(--shadow-2);
            padding: var(--space-6);
            margin-bottom: var(--space-6);
            display: flex;
            align-items: center;
            justify-content: space-between;
            flex-wrap: wrap;
            gap: var(--space-4);
          "
        >
          <div>
            <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: 4px;">
              <h1
                style="
                  font-family: var(--font-heading);
                  font-size: var(--text-2xl);
                  font-weight: 700;
                  color: var(--color-foreground);
                  margin: 0;
                "
              >
                {{ pasien()!.nama }}
              </h1>
              <span
                style="
                  font-family: var(--font-body);
                  font-size: var(--text-xs);
                  padding: 2px 8px;
                  border-radius: var(--radius-full);
                  background: var(--color-muted);
                  color: var(--color-muted-foreground);
                  font-weight: 600;
                "
              >
                ID Pasien #{{ pasien()!.id }}
              </span>
            </div>
            <p
              style="
                font-family: var(--font-body);
                font-size: var(--text-sm);
                color: var(--color-muted-foreground);
                margin: 0;
              "
            >
              Versi Data: v{{ pasien()!.version }}
            </p>
          </div>

          <a
            [routerLink]="['/pasien', pasien()!.id, 'edit']"
            class="kl-btn-secondary"
            style="
              text-decoration: none;
              display: inline-flex;
              align-items: center;
              gap: var(--space-2);
            "
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
        </div>

        <!-- Grid layout: Biodata + Riwayat Ringkas -->
        <div
          style="
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
            gap: var(--space-6);
          "
        >
          <!-- ── Card 1: Biodata Lengkap ── -->
          <div
            style="
              background: var(--color-card);
              border: 1px solid var(--color-border);
              border-radius: var(--radius-md);
              box-shadow: var(--shadow-2);
              padding: var(--space-6);
            "
          >
            <h2
              style="
                font-family: var(--font-heading);
                font-size: var(--text-lg);
                font-weight: 700;
                color: var(--color-foreground);
                margin-bottom: var(--space-4);
                padding-bottom: var(--space-2);
                border-bottom: 1px solid var(--color-border);
              "
            >
              Biodata Lengkap
            </h2>

            <dl
              style="
                display: grid;
                grid-template-columns: 140px 1fr;
                gap: 12px 16px;
                font-family: var(--font-body);
                font-size: var(--text-sm);
                margin: 0;
              "
            >
              <dt style="color: var(--color-muted-foreground); font-weight: 500;">NIK</dt>
              <dd style="margin: 0; font-weight: 600;">
                @if (pasien()!.nik) {
                  <app-sensitive-value
                    mode="display"
                    [displayValue]="pasien()!.nik!"
                  />
                } @else {
                  <span style="color: var(--color-muted-foreground); font-style: italic;">
                    (Tidak ada NIK)
                  </span>
                }
              </dd>

              <dt style="color: var(--color-muted-foreground); font-weight: 500;">Jenis Kelamin</dt>
              <dd style="margin: 0;">
                {{ pasien()!.jenisKelamin === 'L' ? 'Laki-laki' : 'Perempuan' }}
              </dd>

              <dt style="color: var(--color-muted-foreground); font-weight: 500;">Tanggal Lahir</dt>
              <dd style="margin: 0;">
                {{ formatDate(pasien()!.tanggalLahir) }}
              </dd>

              <dt style="color: var(--color-muted-foreground); font-weight: 500;">No. Telepon</dt>
              <dd style="margin: 0;">
                {{ pasien()!.noTelp }}
              </dd>

              <dt style="color: var(--color-muted-foreground); font-weight: 500;">Alamat</dt>
              <dd style="margin: 0; white-space: pre-wrap;">
                {{ pasien()!.alamat }}
              </dd>

              <dt style="color: var(--color-muted-foreground); font-weight: 500;">Status Consent</dt>
              <dd style="margin: 0;">
                @if (pasien()!.consent) {
                  <span
                    style="
                      color: var(--color-accent);
                      font-weight: 600;
                      display: inline-flex;
                      align-items: center;
                      gap: 4px;
                    "
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                    Disetujui
                  </span>
                } @else {
                  <span style="color: var(--color-destructive); font-weight: 600;">
                    Belum disetujui
                  </span>
                }
              </dd>
            </dl>
          </div>

          <!-- ── Card 2: Riwayat Kunjungan Ringkas ── -->
          <div
            style="
              background: var(--color-card);
              border: 1px solid var(--color-border);
              border-radius: var(--radius-md);
              box-shadow: var(--shadow-2);
              padding: var(--space-6);
            "
          >
            <h2
              style="
                font-family: var(--font-heading);
                font-size: var(--text-lg);
                font-weight: 700;
                color: var(--color-foreground);
                margin-bottom: var(--space-4);
                padding-bottom: var(--space-2);
                border-bottom: 1px solid var(--color-border);
              "
            >
              Riwayat Kunjungan Ringkas
            </h2>

            @if (pasien()!.riwayatKunjunganRingkas && pasien()!.riwayatKunjunganRingkas.length > 0) {
              <ul style="list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 8px;">
                @for (kunjungan of pasien()!.riwayatKunjunganRingkas; track kunjungan.kunjunganId) {
                  <li
                    style="
                      display: flex;
                      align-items: center;
                      justify-content: space-between;
                      padding: 10px 12px;
                      background: var(--color-muted);
                      border-radius: var(--radius-sm);
                      font-family: var(--font-body);
                      font-size: var(--text-sm);
                    "
                  >
                    <div>
                      <span style="font-weight: 600; color: var(--color-foreground);">
                        Kunjungan #{{ kunjungan.kunjunganId }}
                      </span>
                      <span style="display: block; font-size: var(--text-xs); color: var(--color-muted-foreground);">
                        {{ formatDate(kunjungan.tanggal) }}
                      </span>
                    </div>

                    <!-- Status badge (DESIGN.md §2 status colors) -->
                    <span
                      [style]="getStatusBadgeStyle(kunjungan.status)"
                    >
                      {{ getStatusLabel(kunjungan.status) }}
                    </span>
                  </li>
                }
              </ul>
            } @else {
              <div
                style="
                  padding: var(--space-8);
                  text-align: center;
                  color: var(--color-muted-foreground);
                  font-family: var(--font-body);
                  font-size: var(--text-sm);
                "
              >
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

  readonly pasien = signal<Pasien | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  ngOnInit(): void {
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

  getStatusBadgeStyle(status: string): string {
    const base = 'font-size:12px; font-weight:600; padding:2px 8px; border-radius:var(--radius-full);';
    switch (status) {
      case 'menunggu':
        return `${base} background:var(--color-muted); color:var(--color-warning-foreground); border:1px solid var(--color-warning);`;
      case 'dipanggil':
        return `${base} background:var(--color-primary); color:var(--color-primary-foreground);`;
      case 'selesai':
        return `${base} background:var(--color-accent); color:var(--color-accent-foreground);`;
      case 'tidak_hadir':
        return `${base} background:var(--color-muted); color:var(--color-muted-foreground); border:1px solid var(--color-border);`;
      default:
        return `${base} background:var(--color-muted); color:var(--color-muted-foreground);`;
    }
  }
}
