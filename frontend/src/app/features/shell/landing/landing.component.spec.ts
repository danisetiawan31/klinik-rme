import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AuthService } from '../../../core/auth/auth.service';
import { UserResponse } from '../../../core/auth/auth.types';
import { KlinikService } from '../../../core/klinik/klinik.service';
import { KlinikResponse } from '../../../core/klinik/klinik.types';
import { getJakartaYesterdayISODate } from '../../../core/utils/date.utils';
import { AntrianService } from '../../antrian/antrian.service';
import { KunjunganListItem } from '../../antrian/antrian.types';
import { LaporanService } from '../../laporan/laporan.service';
import { LaporanHarian } from '../../laporan/laporan.types';
import { LandingComponent } from './landing.component';

describe('LandingComponent', () => {
  let component: LandingComponent;
  let fixture: ComponentFixture<LandingComponent>;
  let userSignal: WritableSignal<UserResponse | null>;
  let authServiceSpy: { currentUser: WritableSignal<UserResponse | null> };
  let antrianServiceSpy: { getAntrian: ReturnType<typeof vi.fn> };
  let klinikSignal: WritableSignal<KlinikResponse | null>;
  let klinikServiceSpy: {
    klinikInfo: WritableSignal<KlinikResponse | null>;
    isKlinikBuka: ReturnType<typeof vi.fn>;
    fetchKlinikInfo: ReturnType<typeof vi.fn>;
  };
  let laporanServiceSpy: { getLaporanHarian: ReturnType<typeof vi.fn> };

  const mockAntrian: KunjunganListItem[] = [
    {
      id: 1,
      pasienNama: 'Andi Pratama',
      nomorAntrian: 1,
      status: 'dipanggil',
      isPriority: true,
      priorityReason: 'Lansia',
      skipCount: 0,
    },
    {
      id: 2,
      pasienNama: 'Siti Aisyah',
      nomorAntrian: 2,
      status: 'menunggu',
      isPriority: false,
      skipCount: 0,
    },
    {
      id: 3,
      pasienNama: 'Budi Handoko',
      nomorAntrian: 3,
      status: 'selesai',
      isPriority: false,
      skipCount: 0,
    },
  ];

  const mockLaporanHariIni: LaporanHarian = {
    tanggal: '2026-08-14',
    totalKunjungan: 10,
    totalSelesai: 8,
    totalTidakHadir: 1,
  };

  const mockLaporanKemarin: LaporanHarian = {
    tanggal: '2026-08-13',
    totalKunjungan: 8,
    totalSelesai: 7,
    totalTidakHadir: 0,
  };

  beforeEach(async () => {
    userSignal = signal<UserResponse | null>(null);
    authServiceSpy = { currentUser: userSignal };
    antrianServiceSpy = { getAntrian: vi.fn().mockReturnValue(of(mockAntrian)) };
    klinikSignal = signal<KlinikResponse | null>({
      id: 1,
      nama: 'Klinik Pratama Sehat',
      jamBuka: '08:00',
      jamTutup: '20:00',
      isBuka: true,
    });
    klinikServiceSpy = {
      klinikInfo: klinikSignal,
      isKlinikBuka: vi.fn().mockReturnValue(true),
      fetchKlinikInfo: vi.fn().mockReturnValue(of(klinikSignal())),
    };
    laporanServiceSpy = {
      getLaporanHarian: vi.fn().mockImplementation((tanggal?: string) => {
        const yesterday = getJakartaYesterdayISODate();
        if (tanggal === yesterday) {
          return of(mockLaporanKemarin);
        }
        return of(mockLaporanHariIni);
      }),
    };

    await TestBed.configureTestingModule({
      imports: [LandingComponent],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authServiceSpy },
        { provide: AntrianService, useValue: antrianServiceSpy },
        { provide: KlinikService, useValue: klinikServiceSpy },
        { provide: LaporanService, useValue: laporanServiceSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LandingComponent);
    component = fixture.componentInstance;
  });

  it('should render shortcuts and summary metrics correctly for role dokter', () => {
    userSignal.set({ id: 2, nama: 'dr. Budi Santoso', roles: ['dokter'] });
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Selamat datang kembali');
    expect(compiled.textContent).toContain('dr. Budi Santoso!');

    // Check summary metrics computed correctly from mockAntrian
    expect(component.totalPasien()).toBe(3);
    expect(component.antrianMenunggu()).toBe(1);
    expect(component.selesaiDilayani()).toBe(1);
    expect(component.pasienPrioritas()).toBe(1);

    const labels = component.shortcuts().map((s) => s.label);
    expect(labels).toContain('Antrian Pasien');
    expect(labels).toContain('Rekam Medis');
    expect(labels).toContain('Riwayat Pasien');
    expect(labels).toContain('Laporan Harian');

    // Negative assertions
    expect(labels).not.toContain('Manajemen Staff');
    expect(labels).not.toContain('Audit Log System');
  });

  it('should render shortcuts correctly for role petugas', () => {
    userSignal.set({ id: 1, nama: 'Siti Rahmawati', roles: ['petugas'] });
    fixture.detectChanges();

    const labels = component.shortcuts().map((s) => s.label);
    expect(labels).toContain('Pendaftaran Pasien');
    expect(labels).toContain('Kelola Antrian');
    expect(labels).toContain('Laporan Harian');

    expect(labels).not.toContain('Rekam Medis');
    expect(labels).not.toContain('Manajemen Staff');
  });

  it('should render shortcuts correctly for role admin', () => {
    userSignal.set({ id: 3, nama: 'Super Admin', roles: ['admin'] });
    fixture.detectChanges();

    const labels = component.shortcuts().map((s) => s.label);
    expect(labels).toContain('Data Pasien');
    expect(labels).toContain('Daftar Antrian');
    expect(labels).toContain('Manajemen Staff');
    expect(labels).toContain('Audit Log System');
    expect(labels).toContain('Pengaturan Klinik');
  });

  it('should compute performance metrics and positive trend correctly from LaporanService', () => {
    userSignal.set({ id: 2, nama: 'dr. Budi Santoso', roles: ['dokter'] });
    fixture.detectChanges();

    expect(component.totalKunjunganLaporan()).toBe(10);
    expect(component.totalSelesaiLaporan()).toBe(8);
    expect(component.totalTidakHadirLaporan()).toBe(1);
    expect(component.performanceRate()).toBe(80); // 8/10 * 100
    expect(component.attendanceRate()).toBe(90); // (10-1)/10 * 100 = 90%
    expect(component.trendVsKemarin().text).toBe('+25% vs kemarin'); // (10-8)/8 * 100 = 25%
    expect(component.trendVsKemarin().isPositive).toBe(true);
  });

  it('should handle zero baseline yesterday data cleanly in trend computation', () => {
    component.laporanHariIni.set({
      tanggal: '2026-08-14',
      totalKunjungan: 5,
      totalSelesai: 4,
      totalTidakHadir: 0,
    });
    component.laporanKemarin.set({
      tanggal: '2026-08-13',
      totalKunjungan: 0,
      totalSelesai: 0,
      totalTidakHadir: 0,
    });

    expect(component.trendVsKemarin().text).toBe('– Data Awal');
    expect(component.trendVsKemarin().isPositive).toBeNull();
  });
});
