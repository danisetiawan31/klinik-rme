import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthService } from '../../../../core/auth/auth.service';
import { AntrianService } from '../../../antrian/antrian.service';
import { KunjunganDetail } from '../../../antrian/antrian.types';
import { PasienService } from '../../../pasien/pasien.service';
import { Pasien } from '../../../pasien/pasien.types';
import { RekamMedisService } from '../../rekam-medis.service';
import { RekamMedis, RiwayatRekamMedisItem } from '../../rekam-medis.types';
import { RekamMedisFormComponent } from './rekam-medis-form.component';

describe('RekamMedisFormComponent', () => {
  let component: RekamMedisFormComponent;
  let fixture: ComponentFixture<RekamMedisFormComponent>;

  const mockKunjungan: KunjunganDetail = {
    id: 5,
    pasienId: 10,
    nomorAntrian: 3,
    status: 'dipanggil',
    isPriority: true,
    dokterId: 2,
    dipanggilAt: '2026-08-19T09:00:00Z',
  };

  const mockPasien: Pasien = {
    id: 10,
    nik: '3201010101010001',
    nama: 'Budi Santoso',
    tanggalLahir: '1990-05-15',
    jenisKelamin: 'L',
    alamat: 'Jl. Merdeka No. 10',
    noTelp: '08123456789',
    consent: true,
    riwayatKunjunganRingkas: [],
    version: 1,
  };

  const mockRiwayat: RiwayatRekamMedisItem[] = [
    {
      kunjunganId: 1,
      tanggal: '2026-08-10',
      rekamMedis: {
        id: 100,
        keluhan: 'Batuk',
        hasilPemeriksaan: 'Faring normal',
        diagnosis: [{ id: 1, kodeIcd: 'R05', deskripsi: 'Cough' }],
        tindakan: [{ id: 1, jenis: 'resep', deskripsi: 'OBH syr' }],
        createdAt: '2026-08-10T08:30:00Z',
      },
    },
  ];

  let antrianServiceMock: { getKunjungan: any };
  let pasienServiceMock: { getById: any };
  let rekamMedisServiceMock: { getRiwayatByPasien: any; createRekamMedis: any };
  let authServiceMock: { currentUser: any };
  let routerMock: { navigate: any };

  beforeEach(async () => {
    antrianServiceMock = {
      getKunjungan: vi.fn().mockReturnValue(of(mockKunjungan)),
    };
    pasienServiceMock = {
      getById: vi.fn().mockReturnValue(of(mockPasien)),
    };
    rekamMedisServiceMock = {
      getRiwayatByPasien: vi.fn().mockReturnValue(of(mockRiwayat)),
      createRekamMedis: vi.fn().mockReturnValue(
        of({
          id: 200,
          keluhan: 'Demam',
          hasilPemeriksaan: 'Suhu 38C',
          diagnosis: [{ id: 5, kodeIcd: 'J00', deskripsi: 'Nasopharyngitis' }],
          tindakan: [],
          createdAt: '2026-08-19T09:30:00Z',
        } as RekamMedis)
      ),
    };
    authServiceMock = {
      currentUser: vi.fn().mockReturnValue({ id: 2, nama: 'dr. Sarah', roles: ['dokter'] }),
    };
    routerMock = {
      navigate: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [RekamMedisFormComponent],
      providers: [
        { provide: AntrianService, useValue: antrianServiceMock },
        { provide: PasienService, useValue: pasienServiceMock },
        { provide: RekamMedisService, useValue: rekamMedisServiceMock },
        { provide: AuthService, useValue: authServiceMock },
        { provide: Router, useValue: routerMock },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'kunjunganId' ? '5' : null),
              },
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RekamMedisFormComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and load kunjungan, pasien, and riwayat data on init', () => {
    expect(component).toBeTruthy();
    expect(antrianServiceMock.getKunjungan).toHaveBeenCalledWith(5);
    expect(pasienServiceMock.getById).toHaveBeenCalledWith(10);
    expect(rekamMedisServiceMock.getRiwayatByPasien).toHaveBeenCalledWith(10);
    expect(component.kunjungan()).toEqual(mockKunjungan);
    expect(component.pasien()).toEqual(mockPasien);
    expect(component.riwayatList()).toEqual(mockRiwayat);
    expect(component.isLoading()).toBe(false);
  });

  it('should have invalid form initially when required fields are empty', () => {
    expect(component.form.valid).toBe(false);
    expect(component.form.get('keluhan')?.valid).toBe(false);
    expect(component.form.get('hasilPemeriksaan')?.valid).toBe(false);
    expect(component.diagnosisArray.length).toBe(1);
    expect(component.diagnosisArray.at(0).valid).toBe(false);
  });

  it('should allow adding and removing diagnosis items in FormArray', () => {
    expect(component.diagnosisArray.length).toBe(1);

    component.addDiagnosis();
    expect(component.diagnosisArray.length).toBe(2);

    component.removeDiagnosis(1);
    expect(component.diagnosisArray.length).toBe(1);

    // Should not remove if only 1 remains
    component.removeDiagnosis(0);
    expect(component.diagnosisArray.length).toBe(1);
  });

  it('should allow adding and removing tindakan items in FormArray', () => {
    expect(component.tindakanArray.length).toBe(0);

    component.addTindakan('resep');
    expect(component.tindakanArray.length).toBe(1);
    expect(component.tindakanArray.at(0).get('jenis')?.value).toBe('resep');

    component.addTindakan('tindakan');
    expect(component.tindakanArray.length).toBe(2);
    expect(component.tindakanArray.at(1).get('jenis')?.value).toBe('tindakan');

    component.removeTindakan(0);
    expect(component.tindakanArray.length).toBe(1);
  });

  it('should render child SOAP subcomponents and patient header correctly', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('app-soap-patient-header')).toBeTruthy();
    expect(compiled.querySelector('app-soap-subjective-section')).toBeTruthy();
    expect(compiled.querySelector('app-soap-objective-section')).toBeTruthy();
    expect(compiled.querySelector('app-soap-assessment-section')).toBeTruthy();
    expect(compiled.querySelector('app-soap-plan-section')).toBeTruthy();
  });

  it('should submit valid rekam medis and navigate to /antrian on success', () => {
    component.form.patchValue({
      keluhan: 'Demam dan sakit kepala sejak kemarin',
      hasilPemeriksaan: 'Suhu 38.5C, nadi 84x/menit, faring hiperemis',
    });

    component.diagnosisArray.at(0).patchValue({
      kodeIcd: 'j00',
      deskripsi: 'Acute nasopharyngitis',
    });

    component.addTindakan('resep');
    component.tindakanArray.at(0).patchValue({
      jenis: 'resep',
      deskripsi: 'Paracetamol 500mg 3x1',
    });

    expect(component.form.valid).toBe(true);

    component.onSubmit();

    expect(rekamMedisServiceMock.createRekamMedis).toHaveBeenCalledWith(5, {
      keluhan: 'Demam dan sakit kepala sejak kemarin',
      hasilPemeriksaan: 'Suhu 38.5C, nadi 84x/menit, faring hiperemis',
      diagnosis: [
        {
          kodeIcd: 'J00',
          deskripsi: 'Acute nasopharyngitis',
        },
      ],
      tindakan: [
        {
          jenis: 'resep',
          deskripsi: 'Paracetamol 500mg 3x1',
        },
      ],
    });

    expect(routerMock.navigate).toHaveBeenCalledWith(['/antrian']);
  });

  it('should handle REKAM_MEDIS_ALREADY_EXISTS 409 error by navigating to detail', () => {
    rekamMedisServiceMock.createRekamMedis = vi.fn().mockReturnValue(
      throwError(() => ({
        error: {
          error: {
            code: 'REKAM_MEDIS_ALREADY_EXISTS',
            message: 'Rekam medis sudah ada',
          },
        },
      }))
    );

    component.form.patchValue({
      keluhan: 'Demam',
      hasilPemeriksaan: 'Suhu 38C',
    });
    component.diagnosisArray.at(0).patchValue({
      deskripsi: 'Demam biasa',
    });

    component.onSubmit();

    expect(routerMock.navigate).toHaveBeenCalledWith(['/rekam-medis/kunjungan', 5]);
  });
});
