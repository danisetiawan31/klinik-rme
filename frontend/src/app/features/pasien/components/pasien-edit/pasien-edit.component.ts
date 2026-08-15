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
import { toast } from '@spartan-ng/brain/sonner';
import { HlmCardImports } from '../../../../shared/ui/card/src/index';
import { PasienService } from '../../pasien.service';
import { nikFormatValidator } from '../../pasien.validators';

@Component({
  selector: 'app-pasien-edit',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, ...HlmCardImports],
  templateUrl: './pasien-edit.component.html',
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
