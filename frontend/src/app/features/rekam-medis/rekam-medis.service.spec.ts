import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { environment } from '../../../environments/environment';
import { RekamMedisService } from './rekam-medis.service';
import {
  CreateAddendumDto,
  CreateRekamMedisDto,
  RekamMedis,
  RiwayatRekamMedisItem,
} from './rekam-medis.types';

describe('RekamMedisService', () => {
  let service: RekamMedisService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), RekamMedisService],
    });
    service = TestBed.inject(RekamMedisService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should fetch leaf rekam medis for a kunjungan via GET /kunjungan/:id/rekam-medis', () => {
    const mockRekamMedis: RekamMedis = {
      id: 1,
      keluhan: 'Demam 3 hari',
      hasilPemeriksaan: 'Suhu 38.5C, faring hiperemis',
      diagnosis: [{ id: 10, kodeIcd: 'J00', deskripsi: 'Nasopharyngitis' }],
      tindakan: [{ id: 20, jenis: 'resep', deskripsi: 'Paracetamol 500mg 3x1' }],
      isAddendum: false,
      createdAt: '2026-08-19T09:00:00Z',
    };

    service.getRekamMedisByKunjungan(5).subscribe((res) => {
      expect(res).toEqual(mockRekamMedis);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/kunjungan/5/rekam-medis`);
    expect(req.request.method).toBe('GET');
    req.flush(mockRekamMedis);
  });

  it('should create rekam medis awal via POST /kunjungan/:id/rekam-medis', () => {
    const payload: CreateRekamMedisDto = {
      keluhan: 'Batuk berdahak',
      hasilPemeriksaan: 'Ronki basah halus minimal',
      diagnosis: [{ kodeIcd: 'J20.9', deskripsi: 'Acute bronchitis, unspecified' }],
      tindakan: [{ jenis: 'resep', deskripsi: 'Ambroxol syr 3x1 cth' }],
    };

    const mockCreated: RekamMedis = {
      id: 2,
      keluhan: payload.keluhan,
      hasilPemeriksaan: payload.hasilPemeriksaan,
      diagnosis: [{ id: 11, kodeIcd: 'J20.9', deskripsi: 'Acute bronchitis, unspecified' }],
      tindakan: [{ id: 21, jenis: 'resep', deskripsi: 'Ambroxol syr 3x1 cth' }],
      isAddendum: false,
      createdAt: '2026-08-19T10:00:00Z',
    };

    service.createRekamMedis(5, payload).subscribe((res) => {
      expect(res).toEqual(mockCreated);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/kunjungan/5/rekam-medis`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(payload);
    req.flush(mockCreated, { status: 201, statusText: 'Created' });
  });

  it('should create addendum koreksi via POST /rekam-medis/:id/addendum', () => {
    const payload: CreateAddendumDto = {
      alasanAddendum: 'Koreksi dosis obat karena berat badan anak',
      keluhan: 'Batuk berdahak anak',
      tindakan: [{ jenis: 'resep', deskripsi: 'Ambroxol drop 3x0.5ml' }],
    };

    const mockAddendum: RekamMedis = {
      id: 3,
      addendumOf: 2,
      keluhan: 'Batuk berdahak anak',
      hasilPemeriksaan: 'Ronki basah halus minimal',
      diagnosis: [{ id: 12, kodeIcd: 'J20.9', deskripsi: 'Acute bronchitis, unspecified' }],
      tindakan: [{ id: 22, jenis: 'resep', deskripsi: 'Ambroxol drop 3x0.5ml' }],
      isAddendum: true,
      createdAt: '2026-08-19T10:30:00Z',
    };

    service.createAddendum(2, payload).subscribe((res) => {
      expect(res).toEqual(mockAddendum);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/rekam-medis/2/addendum`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(payload);
    req.flush(mockAddendum, { status: 201, statusText: 'Created' });
  });

  it('should fetch patient clinical history via GET /pasien/:id/riwayat', () => {
    const mockRiwayat: RiwayatRekamMedisItem[] = [
      {
        kunjunganId: 5,
        tanggal: '2026-08-19',
        rekamMedis: {
          id: 1,
          keluhan: 'Demam',
          hasilPemeriksaan: 'Suhu 38.5C',
          diagnosis: [{ id: 10, kodeIcd: 'R50.9', deskripsi: 'Fever unspecified' }],
          tindakan: [{ id: 20, jenis: 'resep', deskripsi: 'Paracetamol' }],
          createdAt: '2026-08-19T09:00:00Z',
        },
      },
    ];

    service.getRiwayatByPasien(1).subscribe((res) => {
      expect(res).toEqual(mockRiwayat);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/pasien/1/riwayat`);
    expect(req.request.method).toBe('GET');
    req.flush(mockRiwayat);
  });
});
