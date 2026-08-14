import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { environment } from '../../../environments/environment';
import { AntrianService } from './antrian.service';
import {
  CreateKunjunganRequest,
  CreateKunjunganResponse,
  KunjunganListItem,
  PanggilBerikutnyaResponse,
} from './antrian.types';

describe('AntrianService', () => {
  let service: AntrianService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), AntrianService],
    });
    service = TestBed.inject(AntrianService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should create kunjungan queue entry via POST /kunjungan', () => {
    const payload: CreateKunjunganRequest = {
      pasienId: 10,
      isPriority: true,
      priorityReason: 'Lansia',
    };
    const mockResponse: CreateKunjunganResponse = {
      id: 101,
      nomorAntrian: 7,
      status: 'menunggu',
      tanggalKunjungan: '2026-08-14',
    };

    service.create(payload).subscribe((res) => {
      expect(res).toEqual(mockResponse);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/kunjungan`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(payload);
    req.flush(mockResponse, { status: 201, statusText: 'Created' });
  });

  it('should fetch antrian list for given klinikId', () => {
    const mockData: KunjunganListItem[] = [
      {
        id: 1,
        nomorAntrian: 1,
        status: 'menunggu',
        isPriority: false,
        priorityReason: null,
        skipCount: 0,
        pasienNama: 'Budi Santoso',
      },
    ];

    service.getAntrian(1).subscribe((res) => {
      expect(res).toEqual(mockData);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/klinik/1/antrian`);
    expect(req.request.method).toBe('GET');
    req.flush(mockData);
  });

  it('should call panggilBerikutnya and return data on 200 OK', () => {
    const mockResponse: PanggilBerikutnyaResponse = {
      id: 10,
      nomorAntrian: 5,
      pasienNama: 'Ahmad Dahlan',
      dokterId: 2,
      dipanggilAt: '2026-08-14T09:00:00Z',
    };

    service.panggilBerikutnya(1).subscribe((res) => {
      expect(res).toEqual(mockResponse);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/klinik/1/panggil-berikutnya`);
    expect(req.request.method).toBe('POST');
    req.flush(mockResponse, { status: 200, statusText: 'OK' });
  });

  it('should call panggilBerikutnya and return null on 204 No Content', () => {
    service.panggilBerikutnya(1).subscribe((res) => {
      expect(res).toBeNull();
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/klinik/1/panggil-berikutnya`);
    expect(req.request.method).toBe('POST');
    req.flush(null, { status: 204, statusText: 'No Content' });
  });

  it('should call lewati with kunjunganId', () => {
    const mockResponse = { id: 10, status: 'menunggu', skipCount: 1 };

    service.lewati(10).subscribe((res) => {
      expect(res).toEqual(mockResponse);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/kunjungan/10/lewati`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({});
    req.flush(mockResponse);
  });

  it('should call tidakHadir with kunjunganId', () => {
    const mockResponse = { id: 10, status: 'tidak_hadir' };

    service.tidakHadir(10).subscribe((res) => {
      expect(res).toEqual(mockResponse);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/kunjungan/10/tidak-hadir`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({});
    req.flush(mockResponse);
  });
});
