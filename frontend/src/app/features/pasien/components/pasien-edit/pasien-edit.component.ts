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
    <div class="min-h-full p-6 bg-background">
      <!-- Page heading -->
      <div class="mb-6">
        <h1 class="font-heading text-2xl font-bold text-foreground mb-1">
          Edit Biodata Pasien
        </h1>
        <p class="font-sans text-sm text-muted-foreground">
          Perbarui data identitas pasien. Versi Data Saat Ini: <strong>v{{ version() }}</strong>
        </p>
      </div>

      <!-- ── Card Form ── -->
      <div class="bg-card border border-border rounded-md shadow-2 p-6 sm:p-8 max-w-[640px]">
        <!-- 409 Optimistic Lock Hybrid UX Banner -->
        @if (isConflict()) {
          <div
            role="alert"
            aria-live="assertive"
            class="flex items-center justify-between gap-3 p-4 mb-6 bg-muted border border-warning rounded-sm flex-wrap"
          >
            <div class="flex items-center gap-3">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20" height="20" viewBox="0 0 24 24"
                fill="none" stroke="currentColor" stroke-width="2"
                stroke-linecap="round" stroke-linejoin="round"
                class="text-warning shrink-0"
                aria-hidden="true"
              >
                <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                <line x1="12" y1="9" x2="12" y2="13"/>
                <line x1="12" y1="17" x2="12.01" y2="17"/>
              </svg>
              <span class="font-sans text-sm font-semibold text-warning-foreground">
                Data sudah diubah oleh staff lain.
              </span>
            </div>

            <button
              type="button"
              class="kl-btn-secondary text-xs px-3 py-1"
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
          <div class="p-8 text-center text-muted-foreground font-sans">
            <svg
              class="kl-spinner text-primary mx-auto mb-3 block"
              xmlns="http://www.w3.org/2000/svg"
              width="24" height="24" viewBox="0 0 24 24"
              fill="none" stroke="currentColor" stroke-width="2.5"
              stroke-linecap="round" aria-hidden="true"
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
            class="flex flex-col gap-4"
          >
            <!-- ── NIK (nullable) ── -->
            <div class="flex flex-col gap-1.5">
              <label
                for="edit-nik"
                class="font-sans text-sm font-semibold text-foreground"
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
                  class="text-xs text-destructive"
                  role="alert"
                >
                  NIK harus berupa 16 digit angka
                </span>
              }
            </div>

            <!-- ── Nama ── -->
            <div class="flex flex-col gap-1.5">
              <label
                for="edit-nama"
                class="font-sans text-sm font-semibold text-foreground"
              >
                Nama Lengkap <span class="text-destructive">*</span>
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
                  class="text-xs text-destructive"
                  role="alert"
                >
                  Nama wajib diisi
                </span>
              }
            </div>

            <!-- ── Tanggal Lahir ── -->
            <div class="flex flex-col gap-1.5">
              <label
                for="edit-tgl-lahir"
                class="font-sans text-sm font-semibold text-foreground"
              >
                Tanggal Lahir <span class="text-destructive">*</span>
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
                  class="text-xs text-destructive"
                  role="alert"
                >
                  Tanggal lahir wajib diisi
                </span>
              }
            </div>

            <!-- ── Jenis Kelamin ── -->
            <div class="flex flex-col gap-1.5">
              <label
                for="edit-jenis-kelamin"
                class="font-sans text-sm font-semibold text-foreground"
              >
                Jenis Kelamin <span class="text-destructive">*</span>
              </label>
              <select
                id="edit-jenis-kelamin"
                formControlName="jenisKelamin"
                class="kl-input cursor-pointer"
                [attr.aria-invalid]="jenisKelaminCtrl.touched && jenisKelaminCtrl.invalid ? 'true' : null"
              >
                <option value="" disabled>-- Pilih jenis kelamin --</option>
                <option value="L">Laki-laki</option>
                <option value="P">Perempuan</option>
              </select>
              @if (jenisKelaminCtrl.touched && jenisKelaminCtrl.errors?.['required']) {
                <span
                  class="text-xs text-destructive"
                  role="alert"
                >
                  Jenis kelamin wajib dipilih
                </span>
              }
            </div>

            <!-- ── Alamat ── -->
            <div class="flex flex-col gap-1.5">
              <label
                for="edit-alamat"
                class="font-sans text-sm font-semibold text-foreground"
              >
                Alamat <span class="text-destructive">*</span>
              </label>
              <textarea
                id="edit-alamat"
                formControlName="alamat"
                rows="3"
                placeholder="Alamat lengkap pasien"
                class="kl-input resize-y min-h-[72px]"
                [attr.aria-invalid]="alamatCtrl.touched && alamatCtrl.invalid ? 'true' : null"
              ></textarea>
              @if (alamatCtrl.touched && alamatCtrl.errors?.['required']) {
                <span
                  class="text-xs text-destructive"
                  role="alert"
                >
                  Alamat wajib diisi
                </span>
              }
            </div>

            <!-- ── No. Telepon ── -->
            <div class="flex flex-col gap-1.5">
              <label
                for="edit-notelp"
                class="font-sans text-sm font-semibold text-foreground"
              >
                Nomor Telepon <span class="text-destructive">*</span>
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
                  class="text-xs text-destructive"
                  role="alert"
                >
                  Nomor telepon wajib diisi
                </span>
              }
            </div>

            <!-- ── Actions ── -->
            <div class="flex gap-3 justify-end pt-4">
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
