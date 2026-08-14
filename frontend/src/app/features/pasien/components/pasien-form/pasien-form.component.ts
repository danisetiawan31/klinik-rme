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
import { nikFormatValidator } from '../../pasien.validators';

@Component({
  selector: 'app-pasien-form',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, ToastComponent],
  templateUrl: './pasien-form.component.html',
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
