import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { AuthService } from '../../../../core/auth/auth.service';
import { RealtimeService } from '../../../../core/realtime/realtime.service';
import { AntrianService } from '../../../antrian/antrian.service';
import { KunjunganListItem } from '../../../antrian/antrian.types';
import { PasienService } from '../../../pasien/pasien.service';
import { RekamMedisListComponent } from './rekam-medis-list.component';

describe('RekamMedisListComponent', () => {
  let component: RekamMedisListComponent;
  let fixture: ComponentFixture<RekamMedisListComponent>;
  let mockAntrianService: any;
  let mockPasienService: any;
  let mockRealtimeService: any;
  let mockAuthService: any;

  const mockAntrian: KunjunganListItem[] = [
    {
      id: 1,
      nomorAntrian: 1,
      pasienNama: 'Ahmad Pratama',
      status: 'dipanggil',
      isPriority: false,
    },
    {
      id: 2,
      nomorAntrian: 2,
      pasienNama: 'Dewi Lestari',
      status: 'selesai',
      isPriority: true,
      priorityReason: 'Lansia',
    },
  ];

  beforeEach(async () => {
    mockAntrianService = {
      getAntrian: vi.fn().mockReturnValue(of(mockAntrian)),
    };

    mockPasienService = {
      search: vi.fn().mockReturnValue(of({ items: [], totalCount: 0 })),
    };

    mockRealtimeService = {
      lastUpdateAt: signal<number | null>(null),
    };

    mockAuthService = {
      currentUser: vi.fn().mockReturnValue({ id: 2, nama: 'dr. Budi', roles: ['dokter'] }),
    };

    await TestBed.configureTestingModule({
      imports: [RekamMedisListComponent],
      providers: [
        provideRouter([]),
        { provide: AntrianService, useValue: mockAntrianService },
        { provide: PasienService, useValue: mockPasienService },
        { provide: RealtimeService, useValue: mockRealtimeService },
        { provide: AuthService, useValue: mockAuthService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(RekamMedisListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and load antrian on init', () => {
    expect(component).toBeTruthy();
    expect(mockAntrianService.getAntrian).toHaveBeenCalled();
    expect(component.antrianList().length).toBe(2);
  });

  it('should compute activeCallingPatient correctly', () => {
    expect(component.activeCallingPatient()?.id).toBe(1);
    expect(component.activeCallingPatient()?.pasienNama).toBe('Ahmad Pratama');
  });

  it('should format queue numbers with 3-digit zero-padding', () => {
    expect(component.formatQueueNumber(5)).toBe('005');
    expect(component.formatQueueNumber(null)).toBe('---');
  });

  it('should filter list by activeTab correctly', () => {
    component.activeTab.set('dipanggil');
    expect(component.filteredList().length).toBe(1);
    expect(component.filteredList()[0].status).toBe('dipanggil');

    component.activeTab.set('selesai');
    expect(component.filteredList().length).toBe(1);
    expect(component.filteredList()[0].status).toBe('selesai');

    component.activeTab.set('semua');
    expect(component.filteredList().length).toBe(2);
  });
});
