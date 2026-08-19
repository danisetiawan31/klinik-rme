import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthService } from '../../../../core/auth/auth.service';
import { AntrianService } from '../../../antrian/antrian.service';
import { KunjunganDetail } from '../../../antrian/antrian.types';
import { PasienService } from '../../../pasien/pasien.service';
import { Pasien } from '../../../pasien/pasien.types';
import { RekamMedisService } from '../../rekam-medis.service';
import { RekamMedis } from '../../rekam-medis.types';
import { RekamMedisDetailComponent } from './rekam-medis-detail.component';

describe('RekamMedisDetailComponent', () => {
  let component: RekamMedisDetailComponent;
  let fixture: ComponentFixture<RekamMedisDetailComponent>;

  const mockRekamMedis: RekamMedis = {
    id: 10,
    keluhan: 'Batuk berdahak 4 hari',
    hasilPemeriksaan: 'TD 120/80, Suhu 37.8C, Ronki (+)',
    diagnosis: [{ id: 1, kodeIcd: 'J20.9', deskripsi: 'Acute Bronchitis' }],
    tindakan: [{ id: 1, jenis: 'resep', deskripsi: 'Ambroxol 3x1' }],
    isAddendum: false,
    createdAt: '2026-08-19T09:00:00Z',
  };

  const mockKunjungan: KunjunganDetail = {
    id: 5,
    pasienId: 12,
    nomorAntrian: 2,
    status: 'selesai',
    isPriority: false,
    dokterId: 2,
    dipanggilAt: '2026-08-19T08:50:00Z',
  };

  const mockPasien: Pasien = {
    id: 12,
    nik: '3201010101010002',
    nama: 'Siti Aminah',
    tanggalLahir: '1985-08-20',
    jenisKelamin: 'P',
    alamat: 'Jl. Melati No. 5',
    noTelp: '08198765432',
    consent: true,
    riwayatKunjunganRingkas: [],
    version: 1,
  };

  let rekamMedisServiceMock: {
    getRekamMedisByKunjungan: any;
    createAddendum: any;
  };
  let antrianServiceMock: { getKunjungan: any };
  let pasienServiceMock: { getById: any };
  let authServiceMock: { currentUser: any };
  let routerMock: { navigate: any };

  beforeEach(async () => {
    rekamMedisServiceMock = {
      getRekamMedisByKunjungan: vi.fn().mockReturnValue(of(mockRekamMedis)),
      createAddendum: vi.fn().mockReturnValue(
        of({
          ...mockRekamMedis,
          id: 11,
          addendumOf: 10,
          isAddendum: true,
          keluhan: 'Batuk berdahak sudah reda',
        })
      ),
    };
    antrianServiceMock = {
      getKunjungan: vi.fn().mockReturnValue(of(mockKunjungan)),
    };
    pasienServiceMock = {
      getById: vi.fn().mockReturnValue(of(mockPasien)),
    };
    authServiceMock = {
      currentUser: vi.fn().mockReturnValue({ id: 2, nama: 'dr. Sarah', roles: ['dokter'] }),
    };
    routerMock = {
      navigate: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [RekamMedisDetailComponent],
      providers: [
        { provide: RekamMedisService, useValue: rekamMedisServiceMock },
        { provide: AntrianService, useValue: antrianServiceMock },
        { provide: PasienService, useValue: pasienServiceMock },
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

    fixture = TestBed.createComponent(RekamMedisDetailComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and load leaf rekam medis, kunjungan, and pasien data on init', () => {
    expect(component).toBeTruthy();
    expect(rekamMedisServiceMock.getRekamMedisByKunjungan).toHaveBeenCalledWith(5);
    expect(antrianServiceMock.getKunjungan).toHaveBeenCalledWith(5);
    expect(pasienServiceMock.getById).toHaveBeenCalledWith(12);
    expect(component.rekamMedis()).toEqual(mockRekamMedis);
    expect(component.kunjungan()).toEqual(mockKunjungan);
    expect(component.pasien()).toEqual(mockPasien);
    expect(component.isLoading()).toBe(false);
  });

  it('should handle error when rekam medis is not found', () => {
    rekamMedisServiceMock.getRekamMedisByKunjungan = vi.fn().mockReturnValue(
      throwError(() => ({
        error: {
          error: {
            code: 'REKAM_MEDIS_NOT_FOUND',
            message: 'Rekam medis tidak ditemukan',
          },
        },
      }))
    );

    component.loadData(99);
    expect(component.errorMessage()).toBe('Rekam medis tidak ditemukan');
    expect(component.isLoading()).toBe(false);
  });

  it('should open addendum modal and pre-fill form with current rekam medis data', () => {
    component.openAddendumModal();

    expect(component.addendumForm.get('keluhan')?.value).toBe(mockRekamMedis.keluhan);
    expect(component.addendumForm.get('hasilPemeriksaan')?.value).toBe(mockRekamMedis.hasilPemeriksaan);
    expect(component.addendumForm.get('alasanAddendum')?.value).toBe('');
    expect(component.diagnosisArray.length).toBe(1);
    expect(component.diagnosisArray.at(0).get('kodeIcd')?.value).toBe('J20.9');
    expect(component.tindakanArray.length).toBe(1);
    expect(component.tindakanArray.at(0).get('jenis')?.value).toBe('resep');
  });

  it('should allow adding and removing items in Addendum FormArray', () => {
    component.openAddendumModal();

    component.addDiagnosis();
    expect(component.diagnosisArray.length).toBe(2);

    component.removeDiagnosis(1);
    expect(component.diagnosisArray.length).toBe(1);

    component.addTindakan('tindakan');
    expect(component.tindakanArray.length).toBe(2);

    component.removeTindakan(1);
    expect(component.tindakanArray.length).toBe(1);
  });

  it('should submit addendum and update rekamMedis on success', () => {
    component.openAddendumModal();

    component.addendumForm.patchValue({
      alasanAddendum: 'Koreksi evaluasi terapi lanjutan',
      keluhan: 'Batuk berdahak sudah reda',
      hasilPemeriksaan: 'Ronki (-)',
    });

    expect(component.addendumForm.valid).toBe(true);

    component.submitAddendum();

    expect(rekamMedisServiceMock.createAddendum).toHaveBeenCalledWith(10, {
      alasanAddendum: 'Koreksi evaluasi terapi lanjutan',
      keluhan: 'Batuk berdahak sudah reda',
      hasilPemeriksaan: 'Ronki (-)',
      diagnosis: [{ kodeIcd: 'J20.9', deskripsi: 'Acute Bronchitis' }],
      tindakan: [{ jenis: 'resep', deskripsi: 'Ambroxol 3x1' }],
    });

    expect(component.rekamMedis()?.isAddendum).toBe(true);
    expect(component.rekamMedis()?.id).toBe(11);
  });

  it('should handle ADDENDUM_CONFLICT 409 error by reloading data', () => {
    rekamMedisServiceMock.createAddendum = vi.fn().mockReturnValue(
      throwError(() => ({
        error: {
          error: {
            code: 'ADDENDUM_CONFLICT',
            message: 'Versi rekam medis sudah diubah',
          },
        },
      }))
    );

    component.openAddendumModal();
    component.addendumForm.patchValue({
      alasanAddendum: 'Koreksi dosis obat',
    });

    component.submitAddendum();

    expect(rekamMedisServiceMock.getRekamMedisByKunjungan).toHaveBeenCalled();
  });
});
