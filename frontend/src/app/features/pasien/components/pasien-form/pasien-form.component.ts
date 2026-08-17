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
import { toast } from '@spartan-ng/brain/sonner';
import { HlmButton } from '../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../shared/ui/card/src/index';
import { HlmInput } from '../../../../shared/ui/input/src/lib/hlm-input';
import { HlmLabel } from '../../../../shared/ui/label/src/lib/hlm-label';
import { HlmTextarea } from '../../../../shared/ui/textarea/src/lib/hlm-textarea';
import { PasienService } from '../../pasien.service';
import { PasienSearchItem } from '../../pasien.types';
import { nikFormatValidator } from '../../pasien.validators';

@Component({
  selector: 'app-pasien-form',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    HlmButton,
    HlmInput,
    HlmLabel,
    HlmTextarea,
    ...HlmCardImports,
  ],
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

    const trimmed = val.trim();
    if (trimmed.length !== 16 || !/^\d{16}$/.test(trimmed)) {
      return;
    }

    this.isCheckingNik.set(true);
    this.pasienService.searchByNik(trimmed).subscribe({
      next: (results) => {
        this.isCheckingNik.set(false);
        this.nikDuplicateWarning.set(results.length > 0 ? results[0] : null);
      },
      error: () => {
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
          toast.success(`Pasien ${pasien.nama} berhasil didaftarkan`);
          this.router.navigate(['/pasien', pasien.id]);
        },
        error: (err: any) => {
          this.isLoading.set(false);
          this.form.enable();
          const message =
            err?.error?.error?.message ??
            'Gagal mendaftarkan pasien. Silakan coba lagi.';
          this.errorMessage.set(message);
          toast.error(message);
        },
      });
  }

  onCancel(): void {
    this.router.navigate(['/pasien']);
  }
}
