import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { environment } from '../../../environments/environment';
import { LaporanService } from './laporan.service';
import { LaporanHarian } from './laporan.types';

describe('LaporanService', () => {
  let service: LaporanService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), LaporanService],
    });
    service = TestBed.inject(LaporanService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should fetch laporan harian without query param', () => {
    const mockData: LaporanHarian = {
      tanggal: '2026-08-14',
      totalKunjungan: 15,
      totalSelesai: 12,
      totalTidakHadir: 3,
    };

    service.getLaporanHarian().subscribe((res) => {
      expect(res).toEqual(mockData);
    });

    const req = httpTesting.expectOne(`${environment.apiUrl}/laporan/harian`);
    expect(req.request.method).toBe('GET');
    expect(req.request.params.has('tanggal')).toBe(false);
    req.flush(mockData);
  });

  it('should fetch laporan harian with given tanggal query param', () => {
    const mockData: LaporanHarian = {
      tanggal: '2026-08-10',
      totalKunjungan: 20,
      totalSelesai: 18,
      totalTidakHadir: 2,
    };

    service.getLaporanHarian('2026-08-10').subscribe((res) => {
      expect(res).toEqual(mockData);
    });

    const req = httpTesting.expectOne(
      `${environment.apiUrl}/laporan/harian?tanggal=2026-08-10`
    );
    expect(req.request.method).toBe('GET');
    expect(req.request.params.get('tanggal')).toBe('2026-08-10');
    req.flush(mockData);
  });
});
