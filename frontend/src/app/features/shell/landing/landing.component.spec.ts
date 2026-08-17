import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AuthService } from '../../../core/auth/auth.service';
import { UserResponse } from '../../../core/auth/auth.types';
import { AntrianService } from '../../antrian/antrian.service';
import { KunjunganListItem } from '../../antrian/antrian.types';
import { LandingComponent } from './landing.component';

describe('LandingComponent', () => {
  let component: LandingComponent;
  let fixture: ComponentFixture<LandingComponent>;
  let userSignal: WritableSignal<UserResponse | null>;
  let authServiceSpy: { currentUser: WritableSignal<UserResponse | null> };
  let antrianServiceSpy: { getAntrian: ReturnType<typeof vi.fn> };

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

  beforeEach(async () => {
    userSignal = signal<UserResponse | null>(null);
    authServiceSpy = { currentUser: userSignal };
    antrianServiceSpy = { getAntrian: vi.fn().mockReturnValue(of(mockAntrian)) };

    await TestBed.configureTestingModule({
      imports: [LandingComponent],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authServiceSpy },
        { provide: AntrianService, useValue: antrianServiceSpy },
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
});
