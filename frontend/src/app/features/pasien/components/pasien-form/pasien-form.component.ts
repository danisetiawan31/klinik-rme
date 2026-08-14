import {
  ChangeDetectionStrategy,
  Component,
  inject,
  signal,
} from '@angular/core';
import {
  AbstractControl,
  FormBuilder,
  ReactiveFormsModule,
  ValidationErrors,
  ValidatorFn,
  Validators,
} from '@angular/forms';
import { Router } from '@angular/router';
import { ToastComponent } from '../../../../shared/components/toast/toast.component';
import { PasienService } from '../../pasien.service';
import { PasienSearchItem } from '../../pasien.types';

/** Validator: NIK boleh kosong, tapi kalau diisi wajib persis 16 digit angka */
function nikFormatValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    const val: string = control.value ?? '';
    if (!val) return null; // kosong = nullable, valid
    return /^\d{16}$/.test(val) ? null : { nikFormat: true };
  };
}

@Component({
  selector: 'app-pasien-form',
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
    @if (successMessage()) {
      <app-toast
        [message]="successMessage() || ''"
        type="success"
        (dismiss)="successMessage.set(null)"
      />
    }

    <!-- ── Page wrapper (Zona Content — DESIGN.md §1.1) ── -->
    <div
      style="
        min-height:100%;
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
          Registrasi Pasien Baru
        </h1>
        <p
          style="
            font-family: var(--font-body);
            font-size: var(--text-sm);
            color: var(--color-muted-foreground);
          "
        >
          Lengkapi biodata pasien. Kolom bertanda <span style="color:var(--color-destructive)">*</span> wajib diisi.
        </p>
      </div>

      <!-- ── Card form ── -->
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
        <!-- NIK duplicate warning banner (non-blocking) -->
        @if (nikDuplicateWarning()) {
          <div
            role="alert"
            aria-live="polite"
            style="
              display: flex;
              align-items: flex-start;
              gap: var(--space-2);
              padding: var(--space-3) var(--space-4);
              margin-bottom: var(--space-6);
              background-color: var(--color-muted);
              border: 1.5px solid var(--color-warning);
              border-radius: var(--radius-sm);
            "
          >
            <!-- Warning icon -->
            <svg
              xmlns="http://www.w3.org/2000/svg" width="18" height="18"
              viewBox="0 0 24 24" fill="none"
              stroke="var(--color-warning)" stroke-width="2"
              stroke-linecap="round" stroke-linejoin="round"
              style="flex-shrink:0; margin-top:1px;"
              aria-hidden="true"
            >
              <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
              <line x1="12" y1="9" x2="12" y2="13"/>
              <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
            <p
              style="
                font-family: var(--font-body);
                font-size: var(--text-sm);
                color: var(--color-warning-foreground);
                margin: 0;
              "
            >
              NIK sudah terdaftar atas nama:
              <strong>{{ nikDuplicateWarning()!.nama }}</strong>.
              Anda tetap bisa melanjutkan pendaftaran.
            </p>
          </div>
        }

        <form
          [formGroup]="form"
          (ngSubmit)="onSubmit()"
          novalidate
          style="display:flex; flex-direction:column; gap: var(--space-4);"
        >
          <!-- ── NIK (nullable) ── -->
          <div style="display:flex; flex-direction:column; gap:5px;">
            <label
              for="reg-nik"
              style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
            >
              NIK
            </label>
            <input
              id="reg-nik"
              type="text"
              formControlName="nik"
              inputmode="numeric"
              maxlength="16"
              placeholder="Kosongkan jika tidak ada (pasien tanpa NIK)"
              class="kl-input"
              [attr.aria-invalid]="nikCtrl.touched && nikCtrl.invalid ? 'true' : null"
              (input)="onNikInput()"
            />
            @if (isCheckingNik()) {
              <span
                style="font-size:var(--text-xs);color:var(--color-muted-foreground);"
              >
                Memeriksa NIK…
              </span>
            }
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
              for="reg-nama"
              style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
            >
              Nama Lengkap <span style="color:var(--color-destructive)">*</span>
            </label>
            <input
              id="reg-nama"
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
              for="reg-tgl-lahir"
              style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
            >
              Tanggal Lahir <span style="color:var(--color-destructive)">*</span>
            </label>
            <input
              id="reg-tgl-lahir"
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
              for="reg-jenis-kelamin"
              style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
            >
              Jenis Kelamin <span style="color:var(--color-destructive)">*</span>
            </label>
            <select
              id="reg-jenis-kelamin"
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
              for="reg-alamat"
              style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
            >
              Alamat <span style="color:var(--color-destructive)">*</span>
            </label>
            <textarea
              id="reg-alamat"
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
              for="reg-notelp"
              style="font-family:var(--font-body);font-size:var(--text-sm);font-weight:600;color:var(--color-foreground);"
            >
              Nomor Telepon <span style="color:var(--color-destructive)">*</span>
            </label>
            <input
              id="reg-notelp"
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

          <!-- ── Consent ── -->
          <div
            style="
              padding: var(--space-4);
              background: var(--color-muted);
              border: 1px solid var(--color-border);
              border-radius: var(--radius-sm);
            "
          >
            <label
              style="
                display: flex;
                align-items: flex-start;
                gap: var(--space-3);
                cursor: pointer;
              "
            >
              <input
                id="reg-consent"
                type="checkbox"
                formControlName="consent"
                style="
                  width: 18px; height: 18px;
                  flex-shrink: 0;
                  margin-top: 2px;
                  accent-color: var(--color-primary);
                  cursor: pointer;
                "
                [attr.aria-invalid]="consentCtrl.touched && consentCtrl.invalid ? 'true' : null"
              />
              <span
                style="
                  font-family: var(--font-body);
                  font-size: var(--text-sm);
                  color: var(--color-foreground);
                  line-height: 1.6;
                "
              >
                <strong>Persetujuan pengumpulan data pribadi</strong> — Pasien
                menyetujui data identitas dan riwayat kesehatannya dikumpulkan
                serta disimpan oleh klinik untuk keperluan pelayanan medis,
                sesuai ketentuan yang berlaku.
                <span style="color:var(--color-destructive)">*</span>
              </span>
            </label>
            @if (consentCtrl.touched && consentCtrl.errors?.['required']) {
              <p
                style="
                  font-size:var(--text-xs);
                  color:var(--color-destructive);
                  margin: var(--space-2) 0 0 calc(18px + var(--space-3));
                "
                role="alert"
              >
                Persetujuan wajib diberikan sebelum pendaftaran dapat dilanjutkan
              </p>
            }
          </div>

          <!-- ── Actions ── -->
          <div
            style="
              display: flex;
              gap: var(--space-3);
              justify-content: flex-end;
              padding-top: var(--space-2);
            "
          >
            <button
              type="button"
              class="kl-btn-secondary"
              [disabled]="isLoading()"
              (click)="onCancel()"
            >
              Batal
            </button>
            <button
              type="submit"
              class="kl-btn-primary"
              [disabled]="isLoading()"
              [attr.aria-busy]="isLoading() ? 'true' : null"
            >
              @if (isLoading()) {
                <svg
                  class="kl-spinner" xmlns="http://www.w3.org/2000/svg"
                  width="16" height="16" viewBox="0 0 24 24"
                  fill="none" stroke="currentColor" stroke-width="2.5"
                  stroke-linecap="round" aria-hidden="true"
                >
                  <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                </svg>
              }
              Daftarkan Pasien
            </button>
          </div>
        </form>
      </div>
    </div>
  `,
})
export class PasienFormComponent {
  private fb = inject(FormBuilder);
  private pasienService = inject(PasienService);
  private router = inject(Router);

  readonly isLoading = signal<boolean>(false);
  readonly isCheckingNik = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);
  readonly successMessage = signal<string | null>(null);
  readonly nikDuplicateWarning = signal<PasienSearchItem | null>(null);

  readonly form = this.fb.group({
    nik: ['', nikFormatValidator()],
    nama: ['', Validators.required],
    tanggalLahir: ['', Validators.required],
    jenisKelamin: ['', Validators.required],
    alamat: ['', Validators.required],
    noTelp: ['', Validators.required],
    consent: [false, Validators.requiredTrue],
  });

  get nikCtrl() { return this.form.controls.nik; }
  get namaCtrl() { return this.form.controls.nama; }
  get tanggalLahirCtrl() { return this.form.controls.tanggalLahir; }
  get jenisKelaminCtrl() { return this.form.controls.jenisKelamin; }
  get alamatCtrl() { return this.form.controls.alamat; }
  get noTelpCtrl() { return this.form.controls.noTelp; }
  get consentCtrl() { return this.form.controls.consent; }

  /** Triggered on every NIK keystroke; fires duplicate check only when exactly 16 digits */
  onNikInput(): void {
    const val = this.nikCtrl.value ?? '';
    this.nikDuplicateWarning.set(null);

    if (!/^\d{16}$/.test(val)) return;

    this.isCheckingNik.set(true);
    this.pasienService.searchByNik(val).subscribe({
      next: (results) => {
        this.isCheckingNik.set(false);
        this.nikDuplicateWarning.set(results.length > 0 ? results[0] : null);
      },
      error: () => {
        // Silently ignore duplicate-check errors — non-critical, non-blocking
        this.isCheckingNik.set(false);
      },
    });
  }

  onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const val = this.form.getRawValue();

    this.isLoading.set(true);
    this.errorMessage.set(null);
    this.form.disable();

    this.pasienService
      .create({
        nik: val.nik?.trim() || null,
        nama: val.nama!.trim(),
        tanggalLahir: val.tanggalLahir!,
        jenisKelamin: val.jenisKelamin as 'L' | 'P',
        alamat: val.alamat!.trim(),
        noTelp: val.noTelp!.trim(),
        consent: val.consent!,
      })
      .subscribe({
        next: (pasien) => {
          this.isLoading.set(false);
          this.form.enable();
          this.router.navigate(['/pasien', pasien.id]);
        },
        error: (err: any) => {
          this.isLoading.set(false);
          this.form.enable();
          const message =
            err?.error?.error?.message ??
            'Gagal mendaftarkan pasien. Silakan coba lagi.';
          this.errorMessage.set(message);
        },
      });
  }

  onCancel(): void {
    this.router.navigate(['/pasien']);
  }
}
