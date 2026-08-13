import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthService } from '../../../core/auth/auth.service';
import { UserResponse } from '../../../core/auth/auth.types';
import { LandingComponent } from './landing.component';

describe('LandingComponent', () => {
  let component: LandingComponent;
  let fixture: ComponentFixture<LandingComponent>;
  let userSignal: WritableSignal<UserResponse | null>;
  let authServiceSpy: { currentUser: WritableSignal<UserResponse | null> };

  beforeEach(async () => {
    userSignal = signal<UserResponse | null>(null);
    authServiceSpy = { currentUser: userSignal };

    await TestBed.configureTestingModule({
      imports: [LandingComponent],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authServiceSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LandingComponent);
    component = fixture.componentInstance;
  });

  it('should render shortcuts correctly for role petugas and exclude role-restricted shortcuts', () => {
    userSignal.set({ id: 1, nama: 'Budi Petugas', roles: ['petugas'] });
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Selamat datang, Budi Petugas!');

    const labels = component.shortcuts().map((s) => s.label);
    expect(labels).toContain('Pendaftaran Pasien');
    expect(labels).toContain('Kelola Antrian');
    expect(labels).toContain('Laporan Harian');

    // Negative assertions
    expect(labels).not.toContain('Rekam Medis');
    expect(labels).not.toContain('Riwayat Pasien');
    expect(labels).not.toContain('Manajemen Staff');
    expect(labels).not.toContain('Audit Log System');
    expect(labels).not.toContain('Pengaturan Klinik');
  });

  it('should render shortcuts correctly for role dokter and exclude admin shortcuts', () => {
    userSignal.set({ id: 2, nama: 'Dr. Ani', roles: ['dokter'] });
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Selamat datang, Dr. Ani!');

    const labels = component.shortcuts().map((s) => s.label);
    expect(labels).toContain('Antrian Pasien');
    expect(labels).toContain('Rekam Medis');
    expect(labels).toContain('Riwayat Pasien');
    expect(labels).toContain('Laporan Harian');

    // Negative assertions
    expect(labels).not.toContain('Manajemen Staff');
    expect(labels).not.toContain('Audit Log System');
    expect(labels).not.toContain('Pengaturan Klinik');
  });

  it('should render shortcuts correctly for role admin and exclude dokter-only shortcuts', () => {
    userSignal.set({ id: 3, nama: 'Super Admin', roles: ['admin'] });
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Selamat datang, Super Admin!');

    const labels = component.shortcuts().map((s) => s.label);
    expect(labels).toContain('Data Pasien');
    expect(labels).toContain('Daftar Antrian');
    expect(labels).toContain('Manajemen Staff');
    expect(labels).toContain('Audit Log System');
    expect(labels).toContain('Pengaturan Klinik');
    expect(labels).toContain('Laporan Harian');

    // Negative assertion
    expect(labels).not.toContain('Rekam Medis');
  });
});
