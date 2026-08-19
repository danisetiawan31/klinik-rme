import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute } from '@angular/router';
import { of, throwError } from 'rxjs';
import { KlinikService } from '../../core/klinik/klinik.service';
import { RealtimeService } from '../../core/realtime/realtime.service';
import { AntrianService } from '../antrian/antrian.service';
import { KunjunganListItem } from '../antrian/antrian.types';
import {
  DISPLAY_TOKEN_STORAGE_KEY,
  PapanAntrianComponent,
} from './papan-antrian.component';

describe('PapanAntrianComponent', () => {
  let component: PapanAntrianComponent;
  let fixture: ComponentFixture<PapanAntrianComponent>;
  let antrianServiceMock: any;
  let realtimeServiceMock: any;
  let klinikServiceMock: any;

  const mockAntrianList: KunjunganListItem[] = [
    {
      id: 1,
      nomorAntrian: 1,
      status: 'menunggu',
      isPriority: false,
      priorityReason: null,
      skipCount: 0,
      pasienNama: 'Pasien Reguler',
    },
    {
      id: 2,
      nomorAntrian: 2,
      status: 'dipanggil',
      isPriority: true,
      priorityReason: 'Lansia',
      skipCount: 0,
      pasienNama: 'Pasien Prioritas',
    },
    {
      id: 3,
      nomorAntrian: 3,
      status: 'selesai',
      isPriority: false,
      priorityReason: null,
      skipCount: 0,
      pasienNama: 'Pasien Selesai',
    },
  ];

  beforeEach(async () => {
    localStorage.clear();
    localStorage.setItem(DISPLAY_TOKEN_STORAGE_KEY, 'stored-token-123');

    antrianServiceMock = {
      getAntrian: vi.fn().mockReturnValue(of(mockAntrianList)),
    };

    realtimeServiceMock = {
      connectionStatus: signal('connected'),
      lastUpdateAt: signal<number | null>(null),
      connect: vi.fn(),
      disconnect: vi.fn(),
    };

    klinikServiceMock = {
      klinikInfo: signal({ id: 1, nama: 'Klinik Pratama Sehat', isBuka: true }),
      fetchKlinikInfo: vi.fn().mockReturnValue(of({ id: 1, nama: 'Klinik Pratama Sehat' })),
    };

    await TestBed.configureTestingModule({
      imports: [PapanAntrianComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AntrianService, useValue: antrianServiceMock },
        { provide: RealtimeService, useValue: realtimeServiceMock },
        { provide: KlinikService, useValue: klinikServiceMock },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              queryParamMap: {
                get: (key: string) => null,
              },
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(PapanAntrianComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('should create and initialize display board with stored token', () => {
    expect(component).toBeTruthy();
    expect(component.displayToken()).toBe('stored-token-123');
    expect(antrianServiceMock.getAntrian).toHaveBeenCalledWith(1, 'stored-token-123');
    expect(realtimeServiceMock.connect).toHaveBeenCalledWith({
      klinikId: 1,
      displayToken: 'stored-token-123',
    });
  });

  it('should compute activeCalling and waitingList correctly', () => {
    expect(component.activeCalling()?.nomorAntrian).toBe(2);
    expect(component.activeCalling()?.isPriority).toBe(true);

    expect(component.waitingList().length).toBe(1);
    expect(component.waitingList()[0].nomorAntrian).toBe(1);

    expect(component.totalSelesai()).toBe(1);
  });

  it('should format queue numbers with 3-digit zero-padding', () => {
    expect(component.formatQueueNumber(7)).toBe('007');
    expect(component.formatQueueNumber(42)).toBe('042');
    expect(component.formatQueueNumber(105)).toBe('105');
    expect(component.formatQueueNumber(null)).toBe('---');
  });

  it('should handle 401 UNAUTHORIZED error by prompting token modal', () => {
    antrianServiceMock.getAntrian = vi.fn().mockReturnValue(
      throwError(() => ({
        status: 401,
        error: { code: 'UNAUTHORIZED', message: 'Token tidak valid' },
      }))
    );

    component.fetchAntrian();

    expect(component.errorMessage()).toBe('Display Token tidak valid atau telah dicabut.');
    expect(component.showTokenModal()).toBe(true);
  });

  it('should save new token from modal and reconnect', () => {
    component.tokenInput.set('new-secret-display-token');
    component.saveTokenFromModal();

    expect(component.displayToken()).toBe('new-secret-display-token');
    expect(localStorage.getItem(DISPLAY_TOKEN_STORAGE_KEY)).toBe('new-secret-display-token');
    expect(component.showTokenModal()).toBe(false);
    expect(realtimeServiceMock.connect).toHaveBeenCalledWith({
      klinikId: 1,
      displayToken: 'new-secret-display-token',
    });
  });

  it('should disconnect realtime service on component destroy', () => {
    fixture.destroy();
    expect(realtimeServiceMock.disconnect).toHaveBeenCalled();
  });
});
