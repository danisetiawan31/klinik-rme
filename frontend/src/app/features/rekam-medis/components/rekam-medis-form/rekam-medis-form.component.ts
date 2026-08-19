import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import {
  FormArray,
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideActivity,
  lucideAlertCircle,
  lucideArrowLeft,
  lucideCheckCircle2,
  lucideFileText,
  lucideHistory,
  lucidePill,
  lucidePlus,
  lucideStethoscope,
  lucideTrash2,
} from '@ng-icons/lucide';
import { toast } from '@spartan-ng/brain/sonner';
import { AuthService } from '../../../../core/auth/auth.service';
import { formatJakartaDate } from '../../../../core/utils/date.utils';
import { PriorityBadgeComponent } from '../../../../shared/components/priority-badge/priority-badge.component';
import { SensitiveValueComponent } from '../../../../shared/components/sensitive-value/sensitive-value.component';
import { HlmAlertImports } from '../../../../shared/ui/alert/src/index';
import { HlmBadge } from '../../../../shared/ui/badge/src/lib/hlm-badge';
import { HlmButton } from '../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../shared/ui/card/src/index';
import { HlmIconDirective } from '../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../../../shared/ui/input/src/lib/hlm-input';
import { HlmLabel } from '../../../../shared/ui/label/src/lib/hlm-label';
import { HlmSkeletonImports } from '../../../../shared/ui/skeleton/src/index';
import { HlmTextarea } from '../../../../shared/ui/textarea/src/lib/hlm-textarea';
import { AntrianService } from '../../../antrian/antrian.service';
import { KunjunganDetail } from '../../../antrian/antrian.types';
import { PasienService } from '../../../pasien/pasien.service';
import { Pasien } from '../../../pasien/pasien.types';
import { RekamMedisService } from '../../rekam-medis.service';
import {
  CreateDiagnosisDto,
  CreateRekamMedisDto,
  CreateTindakanDto,
  RiwayatRekamMedisItem,
} from '../../rekam-medis.types';

@Component({
  selector: 'app-rekam-medis-form',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    RouterLink,
    PriorityBadgeComponent,
    SensitiveValueComponent,
    HlmButton,
    HlmInput,
    HlmLabel,
    HlmTextarea,
    HlmBadge,
    NgIcon,
    HlmIconDirective,
    ...HlmAlertImports,
    ...HlmCardImports,
    ...HlmSkeletonImports,
  ],
  providers: [
    provideIcons({
      lucideStethoscope,
      lucideFileText,
      lucidePlus,
      lucideTrash2,
      lucideHistory,
      lucideCheckCircle2,
      lucideAlertCircle,
      lucideArrowLeft,
      lucidePill,
      lucideActivity,
    }),
  ],
  templateUrl: './rekam-medis-form.component.html',
})
export class RekamMedisFormComponent implements OnInit {
  private fb = inject(FormBuilder);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private rekamMedisService = inject(RekamMedisService);
  private antrianService = inject(AntrianService);
  private pasienService = inject(PasienService);
  private authService = inject(AuthService);

  readonly kunjunganId = signal<number>(0);
  readonly kunjungan = signal<KunjunganDetail | null>(null);
  readonly pasien = signal<Pasien | null>(null);
  readonly riwayatList = signal<RiwayatRekamMedisItem[]>([]);
  readonly isLoading = signal<boolean>(false);
  readonly isSubmitting = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);
  readonly showHistory = signal<boolean>(false);

  formatDate(dateStr?: string): string {
    return dateStr ? formatJakartaDate(dateStr) : '-';
  }

  readonly currentUser = this.authService.currentUser;
  readonly isDokter = computed(() => this.currentUser()?.roles.includes('dokter') ?? false);

  readonly form = this.fb.group({
    keluhan: ['', [Validators.required, Validators.minLength(3)]],
    hasilPemeriksaan: ['', [Validators.required, Validators.minLength(3)]],
    diagnosis: this.fb.array([this.createDiagnosisGroup()]),
    tindakan: this.fb.array([]),
  });

  get diagnosisArray(): FormArray {
    return this.form.get('diagnosis') as FormArray;
  }

  get tindakanArray(): FormArray {
    return this.form.get('tindakan') as FormArray;
  }

  ngOnInit(): void {
    const idParam = this.route.snapshot.paramMap.get('kunjunganId');
    if (!idParam || isNaN(Number(idParam))) {
      this.errorMessage.set('ID Kunjungan tidak valid');
      this.isLoading.set(false);
      return;
    }

    const id = Number(idParam);
    this.kunjunganId.set(id);
    this.loadKunjunganData(id);
  }

  createDiagnosisGroup(kodeIcd: string = '', deskripsi: string = ''): FormGroup {
    return this.fb.group({
      kodeIcd: [kodeIcd],
      deskripsi: [deskripsi, [Validators.required, Validators.minLength(2)]],
    });
  }

  createTindakanGroup(jenis: 'tindakan' | 'resep' = 'resep', deskripsi: string = ''): FormGroup {
    return this.fb.group({
      jenis: [jenis, Validators.required],
      deskripsi: [deskripsi, [Validators.required, Validators.minLength(2)]],
    });
  }

  addDiagnosis(): void {
    this.diagnosisArray.push(this.createDiagnosisGroup());
  }

  removeDiagnosis(index: number): void {
    if (this.diagnosisArray.length > 1) {
      this.diagnosisArray.removeAt(index);
    } else {
      toast.info('Minimal 1 diagnosis wajib dicatat');
    }
  }

  addTindakan(jenis: 'tindakan' | 'resep' = 'resep'): void {
    this.tindakanArray.push(this.createTindakanGroup(jenis));
  }

  removeTindakan(index: number): void {
    this.tindakanArray.removeAt(index);
  }

  toggleHistory(): void {
    this.showHistory.update((prev) => !prev);
  }

  loadKunjunganData(kunjunganId: number): void {
    this.isLoading.set(true);
    this.antrianService.getKunjungan(kunjunganId).subscribe({
      next: (kunj) => {
        this.kunjungan.set(kunj);
        this.loadPasienData(kunj.pasienId);
      },
      error: (err) => {
        this.isLoading.set(false);
        const msg = err?.error?.error?.message || err?.error?.message || 'Gagal memuat data kunjungan';
        this.errorMessage.set(msg);
        toast.error(msg);
      },
    });
  }

  private loadPasienData(pasienId: number): void {
    this.pasienService.getById(pasienId).subscribe({
      next: (p) => {
        this.pasien.set(p);
        this.loadRiwayatMedis(pasienId);
      },
      error: (err) => {
        this.isLoading.set(false);
        const msg = err?.error?.error?.message || err?.error?.message || 'Gagal memuat profil pasien';
        this.errorMessage.set(msg);
        toast.error(msg);
      },
    });
  }

  private loadRiwayatMedis(pasienId: number): void {
    this.rekamMedisService.getRiwayatByPasien(pasienId).subscribe({
      next: (riwayat) => {
        this.riwayatList.set(riwayat);
        this.isLoading.set(false);
      },
      error: () => {
        // Riwayat error tidak memblokir pengisian form RME
        this.isLoading.set(false);
      },
    });
  }

  onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      toast.error('Harap lengkapi semua kolom rekam medis wajib');
      return;
    }

    const kunjId = this.kunjunganId();
    if (!kunjId || this.isSubmitting()) return;

    this.isSubmitting.set(true);

    const formValue = this.form.value;

    const rawDiagnosis = (formValue.diagnosis || []) as Array<{
      kodeIcd?: string | null;
      deskripsi?: string | null;
    }>;
    const diagnosisPayload: CreateDiagnosisDto[] = rawDiagnosis.map((d) => ({
      kodeIcd: d.kodeIcd?.trim() ? d.kodeIcd.trim().toUpperCase() : null,
      deskripsi: d.deskripsi?.trim() || '',
    }));

    const rawTindakan = (formValue.tindakan || []) as Array<{
      jenis?: 'tindakan' | 'resep' | null;
      deskripsi?: string | null;
    }>;
    const tindakanPayload: CreateTindakanDto[] = rawTindakan.map((t) => ({
      jenis: (t.jenis === 'tindakan' ? 'tindakan' : 'resep') as 'tindakan' | 'resep',
      deskripsi: t.deskripsi?.trim() || '',
    }));

    const payload: CreateRekamMedisDto = {
      keluhan: formValue.keluhan?.trim() || '',
      hasilPemeriksaan: formValue.hasilPemeriksaan?.trim() || '',
      diagnosis: diagnosisPayload,
      tindakan: tindakanPayload,
    };

    this.rekamMedisService.createRekamMedis(kunjId, payload).subscribe({
      next: () => {
        this.isSubmitting.set(false);
        toast.success('Rekam medis berhasil disimpan dan kunjungan pasien selesai');
        this.router.navigate(['/antrian']);
      },
      error: (err) => {
        this.isSubmitting.set(false);
        const code = err?.error?.error?.code;
        const msg =
          err?.error?.error?.message ||
          err?.error?.message ||
          'Gagal menyimpan rekam medis pasien';

        if (code === 'REKAM_MEDIS_ALREADY_EXISTS') {
          toast.error('Rekam medis untuk kunjungan ini sudah pernah dibuat sebelumnya');
          this.router.navigate(['/rekam-medis/kunjungan', kunjId]);
        } else {
          toast.error(msg);
        }
      },
    });
  }
}
