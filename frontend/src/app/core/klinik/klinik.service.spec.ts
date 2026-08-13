import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { environment } from '../../../environments/environment';
import { KlinikService } from './klinik.service';
import { KlinikResponse } from './klinik.types';

describe('KlinikService', () => {
  let service: KlinikService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [KlinikService, provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(KlinikService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should fetch klinik info successfully using environment.defaultKlinikId', () => {
    const mockKlinik: KlinikResponse = {
      id: 1,
      nama: 'Klinik Sehat Utama',
      jamBuka: '08:00',
      jamTutup: '16:00',
      isBuka: true,
    };

    service.fetchKlinikInfo().subscribe((res) => {
      expect(res).toEqual(mockKlinik);
      expect(service.klinikInfo()).toEqual(mockKlinik);
    });

    const req = httpMock.expectOne(`${environment.apiUrl}/klinik/${environment.defaultKlinikId}`);
    expect(req.request.method).toBe('GET');
    req.flush(mockKlinik);
  });

  it('should calculate isKlinikBuka correctly when isBuka boolean is provided', () => {
    const mockOpen: KlinikResponse = { id: 1, nama: 'Klinik', isBuka: true };
    const mockClosed: KlinikResponse = { id: 1, nama: 'Klinik', isBuka: false };

    expect(service.isKlinikBuka(mockOpen)).toBe(true);
    expect(service.isKlinikBuka(mockClosed)).toBe(false);
  });

  it('should compute isKlinikBuka anchored to Asia/Jakarta timezone when isBuka is undefined', () => {
    const mockHours: KlinikResponse = {
      id: 1,
      nama: 'Klinik',
      jamBuka: '00:00',
      jamTutup: '23:59',
    };

    expect(service.isKlinikBuka(mockHours)).toBe(true);
  });
});
