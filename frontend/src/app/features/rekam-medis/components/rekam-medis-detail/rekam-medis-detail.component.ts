import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
  viewChild,
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
  lucideAlertTriangle,
  lucideArrowLeft,
  lucideCheckCircle2,
  lucideClock,
  lucideEdit3,
  lucideFileText,
  lucideHistory,
  lucidePill,
  lucidePlus,
  lucideStethoscope,
  lucideTrash2,
  lucideUser,
} from '@ng-icons/lucide';
import { toast } from '@spartan-ng/brain/sonner';
import { AuthService } from '../../../../core/auth/auth.service';
import { PriorityBadgeComponent } from '../../../../shared/components/priority-badge/priority-badge.component';
import { SensitiveValueComponent } from '../../../../shared/components/sensitive-value/sensitive-value.component';
import { HlmAlertImports } from '../../../../shared/ui/alert/src/index';
import { HlmBadge } from '../../../../shared/ui/badge/src/lib/hlm-badge';
import { HlmButton } from '../../../../shared/ui/button/src/lib/hlm-button';
import { HlmCardImports } from '../../../../shared/ui/card/src/index';
import { HlmDialog } from '../../../../shared/ui/dialog/src/lib/hlm-dialog';
import { HlmDialogImports } from '../../../../shared/ui/dialog/src/index';
import { HlmEmptyImports } from '../../../../shared/ui/empty/src/index';
import { HlmIconDirective } from '../../../../shared/ui/icon/src/lib/hlm-icon.directive';
import { HlmInput } from '../../../../shared/ui/input/src/lib/hlm-input';
import { HlmLabel } from '../../../../shared/ui/label/src/lib/hlm-label';
import { HlmSkeletonImports } from '../../../../shared/ui/skeleton/src/index';
import { HlmTableImports } from '../../../../shared/ui/table/src/index';
import { HlmTextarea } from '../../../../shared/ui/textarea/src/lib/hlm-textarea';
import { AntrianService } from '../../../antrian/antrian.service';
import { KunjunganDetail } from '../../../antrian/antrian.types';
import { PasienService } from '../../../pasien/pasien.service';
import { Pasien } from '../../../pasien/pasien.types';
import { RekamMedisService } from '../../rekam-medis.service';
import {
  CreateAddendumDto,
  CreateDiagnosisDto,
  CreateTindakanDto,
  RekamMedis,
} from '../../rekam-medis.types';

@Component({
  selector: 'app-rekam-medis-detail',
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
    ...HlmDialogImports,
    ...HlmEmptyImports,
    ...HlmSkeletonImports,
    ...HlmTableImports,
  ],
  providers: [
    provideIcons({
      lucideStethoscope,
      lucideFileText,
      lucidePlus,
      lucideTrash2,
      lucideHistory,
      lucideUser,
      lucideClock,
      lucideCheckCircle2,
      lucideAlertCircle,
      lucideAlertTriangle,
      lucideArrowLeft,
      lucidePill,
      lucideActivity,
      lucideEdit3,
    }),
  ],
  templateUrl: './rekam-medis-detail.component.html',
})
export class RekamMedisDetailComponent implements OnInit {
  private fb = inject(FormBuilder);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private rekamMedisService = inject(RekamMedisService);
  private antrianService = inject(AntrianService);
  private pasienService = inject(PasienService);
  private authService = inject(AuthService);

  readonly addendumDialog = viewChild<HlmDialog>('addendumDialog');

  readonly kunjunganId = signal<number | null>(null);
  readonly kunjungan = signal<KunjunganDetail | null>(null);
  readonly pasien = signal<Pasien | null>(null);
  readonly rekamMedis = signal<RekamMedis | null>(null);

  readonly isLoading = signal<boolean>(true);
  readonly isSubmittingAddendum = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  readonly currentUser = this.authService.currentUser;
  readonly isDokter = computed(() => this.currentUser()?.roles.includes('dokter') ?? false);

  readonly addendumForm = this.fb.group({
    alasanAddendum: ['', [Validators.required, Validators.minLength(5)]],
    keluhan: ['', [Validators.required, Validators.minLength(3)]],
    hasilPemeriksaan: ['', [Validators.required, Validators.minLength(3)]],
    diagnosis: this.fb.array([]),
    tindakan: this.fb.array([]),
  });

  get diagnosisArray(): FormArray {
    return this.addendumForm.get('diagnosis') as FormArray;
  }

  get tindakanArray(): FormArray {
    return this.addendumForm.get('tindakan') as FormArray;
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
    this.loadData(id);
  }

  loadData(kunjunganId: number): void {
    this.isLoading.set(true);
    this.rekamMedisService.getRekamMedisByKunjungan(kunjunganId).subscribe({
      next: (rm) => {
        this.rekamMedis.set(rm);
        this.loadKunjunganAndPasien(kunjunganId);
      },
      error: (err) => {
        this.isLoading.set(false);
        const msg =
          err?.error?.error?.message ||
          err?.error?.message ||
          'Rekam medis untuk kunjungan ini belum dicatat atau tidak ditemukan';
        this.errorMessage.set(msg);
      },
    });
  }

  private loadKunjunganAndPasien(kunjunganId: number): void {
    this.antrianService.getKunjungan(kunjunganId).subscribe({
      next: (kunj) => {
        this.kunjungan.set(kunj);
        this.pasienService.getById(kunj.pasienId).subscribe({
          next: (p) => {
            this.pasien.set(p);
            this.isLoading.set(false);
          },
          error: () => {
            this.isLoading.set(false);
          },
        });
      },
      error: () => {
        this.isLoading.set(false);
      },
    });
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
      toast.info('Minimal 1 diagnosis wajib ada');
    }
  }

  addTindakan(jenis: 'tindakan' | 'resep' = 'resep'): void {
    this.tindakanArray.push(this.createTindakanGroup(jenis));
  }

  removeTindakan(index: number): void {
    this.tindakanArray.removeAt(index);
  }

  openAddendumModal(): void {
    const rm = this.rekamMedis();
    if (!rm) return;

    this.addendumForm.reset();
    this.diagnosisArray.clear();
    this.tindakanArray.clear();

    this.addendumForm.patchValue({
      alasanAddendum: '',
      keluhan: rm.keluhan,
      hasilPemeriksaan: rm.hasilPemeriksaan,
    });

    if (rm.diagnosis && rm.diagnosis.length > 0) {
      for (const d of rm.diagnosis) {
        this.diagnosisArray.push(this.createDiagnosisGroup(d.kodeIcd || '', d.deskripsi));
      }
    } else {
      this.diagnosisArray.push(this.createDiagnosisGroup());
    }

    if (rm.tindakan && rm.tindakan.length > 0) {
      for (const t of rm.tindakan) {
        this.tindakanArray.push(this.createTindakanGroup(t.jenis, t.deskripsi));
      }
    }

    this.addendumDialog()?.open();
  }

  closeAddendumModal(): void {
    this.addendumDialog()?.close();
  }

  submitAddendum(): void {
    if (this.addendumForm.invalid) {
      this.addendumForm.markAllAsTouched();
      toast.error('Harap isi alasan addendum dan seluruh kolom wajib');
      return;
    }

    const rm = this.rekamMedis();
    if (!rm || this.isSubmittingAddendum()) return;

    this.isSubmittingAddendum.set(true);

    const val = this.addendumForm.value;

    const rawDiagnosis = (val.diagnosis || []) as Array<{
      kodeIcd?: string | null;
      deskripsi?: string | null;
    }>;
    const diagnosisPayload: CreateDiagnosisDto[] = rawDiagnosis.map((d) => ({
      kodeIcd: d.kodeIcd?.trim() ? d.kodeIcd.trim().toUpperCase() : null,
      deskripsi: d.deskripsi?.trim() || '',
    }));

    const rawTindakan = (val.tindakan || []) as Array<{
      jenis?: 'tindakan' | 'resep' | null;
      deskripsi?: string | null;
    }>;
    const tindakanPayload: CreateTindakanDto[] = rawTindakan.map((t) => ({
      jenis: (t.jenis === 'tindakan' ? 'tindakan' : 'resep') as 'tindakan' | 'resep',
      deskripsi: t.deskripsi?.trim() || '',
    }));

    const payload: CreateAddendumDto = {
      alasanAddendum: val.alasanAddendum?.trim() || '',
      keluhan: val.keluhan?.trim() || '',
      hasilPemeriksaan: val.hasilPemeriksaan?.trim() || '',
      diagnosis: diagnosisPayload,
      tindakan: tindakanPayload,
    };

    this.rekamMedisService.createAddendum(rm.id, payload).subscribe({
      next: (newRM) => {
        this.isSubmittingAddendum.set(false);
        this.closeAddendumModal();
        toast.success('Addendum rekam medis berhasil dicatat secara resmi');
        this.rekamMedis.set(newRM);
      },
      error: (err) => {
        this.isSubmittingAddendum.set(false);
        const code = err?.error?.error?.code;
        const msg =
          err?.error?.error?.message ||
          err?.error?.message ||
          'Gagal menyimpan addendum rekam medis';

        if (code === 'ADDENDUM_CONFLICT') {
          toast.error('Versi rekam medis ini sudah diperbarui dokter lain. Memuat data terkini...');
          this.closeAddendumModal();
          const kId = this.kunjunganId();
          if (kId) this.loadData(kId);
        } else {
          toast.error(msg);
        }
      },
    });
  }
}
