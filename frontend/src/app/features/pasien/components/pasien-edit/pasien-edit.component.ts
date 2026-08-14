import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import {
  FormBuilder,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { ToastComponent } from '../../../../shared/components/toast/toast.component';
import { PasienService } from '../../pasien.service';
import { nikFormatValidator } from '../../pasien.validators';

@Component({
  selector: 'app-pasien-edit',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, ToastComponent],
  template: `
    @if (errorMessage()) {
      <app-toast
        [message]="errorMessage() || ''"
        type="error"
        (dismiss)="errorMessage.set(null)"
      />
    }

    <!-- ── Page wrapper (Zona Content — DESIGN.md §1.1) ── -->
    <div
      style="
        min-height: 100%;
        padding: var(--space-6);
        background-color: var(--color-background);
      "
    >
      <!-- Page heading -->
      <div style="margin-bottom: var(--space-6);">
        <h1
          style="
            font-family: var(--font-heading);
            font-size: var(--text-2xl);
            font-weight: 700;
            color: var(--color-foreground);
            margin-bottom: var(--space-1);
          "
        >
          Edit Biodata Pasien
        </h1>
        <p
          style="
            font-family: var(--font-body);
            font-size: var(--text-sm);
            color: var(--color-muted-foreground);
          "
        >
          Perbarui data identitas pasien. Versi Data Saat Ini: <strong>v{{ version() }}</strong>
        </p>
      </div>

      <!-- ── Card Form ── -->
      <div
        style="
          background: var(--color-card);
          border: 1px solid var(--color-border);
          border-radius: var(--radius-md);
          box-shadow: var(--shadow-2);
          padding: var(--space-8);
          max-width: 640px;
        "
      >
        <!-- 409 Optimistic Lock Hybrid UX Banner -->
        @if (isConflict()) {
          <div
            role="alert"
            aria-live="assertive"
            style="
              display: flex;
              align-items: center;
              justify-content: space-between;
              gap: var(--space-3);
              padding: var(--space-4);
              margin-bottom: var(--space-6);
              background-color: var(--color-muted);
              border: 1.5px solid var(--color-warning);
              border-radius: var(--radius-sm);
              flex-wrap: wrap;
            "
          >
            <div style="display: flex; align-items: center; gap: var(--space-3);">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20" height="20" viewBox="0 0 24 24"
                fill="none" stroke="var(--color-warning)" stroke-width="2"
                stroke-linecap="round" stroke-linejoin="round"
                aria-hidden="true"
              >
                <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                <line x1="12" y1="9" x2="12" y2="13"/>
                <line x1="12" y1="17" x2="12.01" y2="17"/>
              </svg>
              <span
                style="
                  font-family: var(--font-body);
                  font-size: var(--text-sm);
                  font-weight: 600;
                  color: var(--color-warning-foreground);
                "
              >
                Data sudah diubah oleh staff lain.
              </span>
            </div>

            <button
              type="button"
              class="kl-btn-secondary"
              style="font-size: var(--text-xs); padding: 4px 12px;"
              [disabled]="isRefetching()"
              (click)="onReloadLatest()"
            >
              @if (isRefetching()) {
                Memuat...
              } @else {
                Muat versi terbaru
              }
            </button>
          </div>
        }

        @if (isLoadingInitial()) {
          <div
            style="
              padding: var(--space-8);
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
            Memuat data pasien...
          </div>
        } @else {
          <form
            [formGroup]="form"
            (ngSubmit)="onSubmit()"
            novalidate
            style="display:flex; flex-direction:column; gap: var(--space-4);"
          >
            <!-- ── NIK (nullable) ── -->
            <div style="display:flex; flex-direction:column; gap:5px;">
              <label
                for="edit-nik"
                style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
              >
                NIK
              </label>
              <input
                id="edit-nik"
                type="text"
                formControlName="nik"
                inputmode="numeric"
                maxlength="16"
                placeholder="Kosongkan jika tidak ada"
                class="kl-input"
                [attr.aria-invalid]="nikCtrl.touched && nikCtrl.invalid ? 'true' : null"
              />
              @if (nikCtrl.touched && nikCtrl.errors?.['nikFormat']) {
                <span
                  style="font-size:var(--text-xs);color:var(--color-destructive);"
                  role="alert"
                >
                  NIK harus berupa 16 digit angka
                </span>
              }
            </div>

            <!-- ── Nama ── -->
            <div style="display:flex; flex-direction:column; gap:5px;">
              <label
                for="edit-nama"
                style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
              >
                Nama Lengkap <span style="color:var(--color-destructive)">*</span>
              </label>
              <input
                id="edit-nama"
                type="text"
                formControlName="nama"
                placeholder="Nama lengkap sesuai identitas"
                class="kl-input"
                [attr.aria-invalid]="namaCtrl.touched && namaCtrl.invalid ? 'true' : null"
              />
              @if (namaCtrl.touched && namaCtrl.errors?.['required']) {
                <span
                  style="font-size:var(--text-xs);color:var(--color-destructive);"
                  role="alert"
                >
                  Nama wajib diisi
                </span>
              }
            </div>

            <!-- ── Tanggal Lahir ── -->
            <div style="display:flex; flex-direction:column; gap:5px;">
              <label
                for="edit-tgl-lahir"
                style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
              >
                Tanggal Lahir <span style="color:var(--color-destructive)">*</span>
              </label>
              <input
                id="edit-tgl-lahir"
                type="date"
                formControlName="tanggalLahir"
                class="kl-input"
                [attr.aria-invalid]="tanggalLahirCtrl.touched && tanggalLahirCtrl.invalid ? 'true' : null"
              />
              @if (tanggalLahirCtrl.touched && tanggalLahirCtrl.errors?.['required']) {
                <span
                  style="font-size:var(--text-xs);color:var(--color-destructive);"
                  role="alert"
                >
                  Tanggal lahir wajib diisi
                </span>
              }
            </div>

            <!-- ── Jenis Kelamin ── -->
            <div style="display:flex; flex-direction:column; gap:5px;">
              <label
                for="edit-jenis-kelamin"
                style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
              >
                Jenis Kelamin <span style="color:var(--color-destructive)">*</span>
              </label>
              <select
                id="edit-jenis-kelamin"
                formControlName="jenisKelamin"
                class="kl-input"
                style="cursor:pointer;"
                [attr.aria-invalid]="jenisKelaminCtrl.touched && jenisKelaminCtrl.invalid ? 'true' : null"
              >
                <option value="" disabled>-- Pilih jenis kelamin --</option>
                <option value="L">Laki-laki</option>
                <option value="P">Perempuan</option>
              </select>
              @if (jenisKelaminCtrl.touched && jenisKelaminCtrl.errors?.['required']) {
                <span
                  style="font-size:var(--text-xs);color:var(--color-destructive);"
                  role="alert"
                >
                  Jenis kelamin wajib dipilih
                </span>
              }
            </div>

            <!-- ── Alamat ── -->
            <div style="display:flex; flex-direction:column; gap:5px;">
              <label
                for="edit-alamat"
                style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
              >
                Alamat <span style="color:var(--color-destructive)">*</span>
              </label>
              <textarea
                id="edit-alamat"
                formControlName="alamat"
                rows="3"
                placeholder="Alamat lengkap pasien"
                class="kl-input"
                style="resize:vertical; min-height:72px;"
                [attr.aria-invalid]="alamatCtrl.touched && alamatCtrl.invalid ? 'true' : null"
              ></textarea>
              @if (alamatCtrl.touched && alamatCtrl.errors?.['required']) {
                <span
                  style="font-size:var(--text-xs);color:var(--color-destructive);"
                  role="alert"
                >
                  Alamat wajib diisi
                </span>
              }
            </div>

            <!-- ── No. Telepon ── -->
            <div style="display:flex; flex-direction:column; gap:5px;">
              <label
                for="edit-notelp"
                style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
              >
                Nomor Telepon <span style="color:var(--color-destructive)">*</span>
              </label>
              <input
                id="edit-notelp"
                type="tel"
                formControlName="noTelp"
                inputmode="tel"
                placeholder="Contoh: 08123456789"
                class="kl-input"
                [attr.aria-invalid]="noTelpCtrl.touched && noTelpCtrl.invalid ? 'true' : null"
              />
              @if (noTelpCtrl.touched && noTelpCtrl.errors?.['required']) {
                <span
                  style="font-size:var(--text-xs);color:var(--color-destructive);"
                  role="alert"
                >
                  Nomor telepon wajib diisi
                </span>
              }
            </div>

            <!-- ── Actions ── -->
            <div
              style="
                display: flex;
                gap: var(--space-3);
                justify-content: flex-end;
                padding-top: var(--space-4);
              "
            >
              <button
                type="button"
                class="kl-btn-secondary"
                [disabled]="isSubmitting()"
                (click)="onCancel()"
              >
                Batal
              </button>
              <button
                type="submit"
                class="kl-btn-primary"
                [disabled]="isSubmitting()"
                [attr.aria-busy]="isSubmitting() ? 'true' : null"
              >
                @if (isSubmitting()) {
                  <svg
                    class="kl-spinner" xmlns="http://www.w3.org/2000/svg"
                    width="16" height="16" viewBox="0 0 24 24"
                    fill="none" stroke="currentColor" stroke-width="2.5"
                    stroke-linecap="round" aria-hidden="true"
                  >
                    <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                  </svg>
                }
                Simpan Perubahan
              </button>
            </div>
          </form>
        }
      </div>
    </div>
  `,
})
export class PasienEditComponent implements OnInit {
  private fb = inject(FormBuilder);
  private pasienService = inject(PasienService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  readonly pasienId = signal<number>(0);
  readonly version = signal<number>(1);

  readonly isLoadingInitial = signal<boolean>(false);
  readonly isSubmitting = signal<boolean>(false);
  readonly isConflict = signal<boolean>(false);
  readonly isRefetching = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  readonly form = this.fb.group({
    nik: ['', nikFormatValidator()],
    nama: ['', Validators.required],
    tanggalLahir: ['', Validators.required],
    jenisKelamin: ['', Validators.required],
    alamat: ['', Validators.required],
    noTelp: ['', Validators.required],
  });

  get nikCtrl() { return this.form.controls.nik; }
  get namaCtrl() { return this.form.controls.nama; }
  get tanggalLahirCtrl() { return this.form.controls.tanggalLahir; }
  get jenisKelaminCtrl() { return this.form.controls.jenisKelamin; }
  get alamatCtrl() { return this.form.controls.alamat; }
  get noTelpCtrl() { return this.form.controls.noTelp; }

  ngOnInit(): void {
    const idParam = this.route.snapshot.paramMap.get('id');
    const id = idParam ? parseInt(idParam, 10) : 0;
    if (!id || isNaN(id)) {
      this.errorMessage.set('ID pasien tidak valid.');
      return;
    }

    this.pasienId.set(id);
    this.fetchData(id);
  }

  fetchData(id: number): void {
    this.isLoadingInitial.set(true);
    this.errorMessage.set(null);

    this.pasienService.getById(id).subscribe({
      next: (pasien) => {
        this.isLoadingInitial.set(false);
        this.version.set(pasien.version);
        this.form.patchValue({
          nik: pasien.nik || '',
          nama: pasien.nama,
          tanggalLahir: pasien.tanggalLahir?.slice(0, 10) || '',
          jenisKelamin: pasien.jenisKelamin,
          alamat: pasien.alamat,
          noTelp: pasien.noTelp,
        });
      },
      error: (err: any) => {
        this.isLoadingInitial.set(false);
        const msg =
          err?.error?.error?.message ??
          'Gagal memuat data pasien. Silakan coba lagi.';
        this.errorMessage.set(msg);
      },
    });
  }

  onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const val = this.form.getRawValue();

    this.isSubmitting.set(true);
    this.errorMessage.set(null);
    this.isConflict.set(false);
    this.form.disable();

    this.pasienService
      .update(this.pasienId(), {
        nik: val.nik?.trim() || null,
        nama: val.nama!.trim(),
        tanggalLahir: val.tanggalLahir!,
        jenisKelamin: val.jenisKelamin as 'L' | 'P',
        alamat: val.alamat!.trim(),
        noTelp: val.noTelp!.trim(),
        version: this.version(),
      })
      .subscribe({
        next: (updatedPasien) => {
          this.isSubmitting.set(false);
          this.form.enable();
          // Navigate back to detail with success toast feedback state
          this.router.navigate(['/pasien', updatedPasien.id], {
            state: { successMessage: 'Biodata pasien berhasil diperbarui.' },
          });
        },
        error: (err: any) => {
          this.isSubmitting.set(false);
          this.form.enable();

          if (err?.status === 409 || err?.error?.error?.code === 'OPTIMISTIC_LOCK_FAILED') {
            // UX 409 Hybrid: Show inline conflict banner, DO NOT reset form fields!
            this.isConflict.set(true);
          } else {
            const message =
              err?.error?.error?.message ??
              'Gagal memperbarui data pasien. Silakan coba lagi.';
            this.errorMessage.set(message);
          }
        },
      });
  }

  onReloadLatest(): void {
    this.isRefetching.set(true);
    this.pasienService.getById(this.pasienId()).subscribe({
      next: (pasien) => {
        this.isRefetching.set(false);
        this.version.set(pasien.version);
        this.form.reset({
          nik: pasien.nik || '',
          nama: pasien.nama,
          tanggalLahir: pasien.tanggalLahir?.slice(0, 10) || '',
          jenisKelamin: pasien.jenisKelamin,
          alamat: pasien.alamat,
          noTelp: pasien.noTelp,
        });
        this.isConflict.set(false);
      },
      error: () => {
        this.isRefetching.set(false);
        this.errorMessage.set('Gagal memuat versi terbaru.');
      },
    });
  }

  onCancel(): void {
    this.router.navigate(['/pasien', this.pasienId()]);
  }
}
