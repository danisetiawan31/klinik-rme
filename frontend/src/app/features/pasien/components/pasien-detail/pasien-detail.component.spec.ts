import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { signal } from '@angular/core';
import { PasienDetailComponent } from './pasien-detail.component';
import { PasienService } from '../../pasien.service';
import { AuthService } from '../../../../core/auth/auth.service';
import { Pasien } from '../../pasien.types';

const mockPasien: Pasien = {
  id: 42,
  nik: '1234567890123456',
  nama: 'Budi Santoso',
  tanggalLahir: '1985-05-15',
  jenisKelamin: 'L',
  alamat: 'Jl. Melati No. 10',
  noTelp: '081234567890',
  consent: true,
  version: 1,
  riwayatKunjunganRingkas: [
    { kunjunganId: 101, tanggal: '2026-08-01', status: 'selesai' },
    { kunjunganId: 102, tanggal: '2026-08-10', status: 'menunggu' },
  ],
};

const mockPasienService = {
  getById: vi.fn(),
};

describe('PasienDetailComponent', () => {
  let component: PasienDetailComponent;
  let fixture: ComponentFixture<PasienDetailComponent>;
  let currentUserSignal = signal<{ id: number; nama: string; roles: string[] } | null>({
    id: 1,
    nama: 'Staff Petugas',
    roles: ['petugas'],
  });

  beforeEach(async () => {
    vi.clearAllMocks();
    currentUserSignal.set({ id: 1, nama: 'Staff Petugas', roles: ['petugas'] });
    mockPasienService.getById.mockReturnValue(of(mockPasien));

    await TestBed.configureTestingModule({
      imports: [PasienDetailComponent],
      providers: [
        provideRouter([]),
        { provide: PasienService, useValue: mockPasienService },
        {
          provide: AuthService,
          useValue: { currentUser: currentUserSignal },
        },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              paramMap: {
                get: (key: string) => (key === 'id' ? '42' : null),
              },
            },
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(PasienDetailComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('fetches patient detail by ID on init', () => {
    expect(mockPasienService.getById).toHaveBeenCalledWith(42);
    expect(component.pasien()).toEqual(mockPasien);
  });

  it('renders patient nama and ID badge', () => {
    const textContent = fixture.nativeElement.textContent;
    expect(textContent).toContain('Budi Santoso');
    expect(textContent).toContain('ID Pasien #42');
  });

  it('renders riwayat kunjungan ringkas correctly', () => {
    const textContent = fixture.nativeElement.textContent;
    expect(textContent).toContain('Kunjungan #101');
    expect(textContent).toContain('Selesai');
    expect(textContent).toContain('Kunjungan #102');
    expect(textContent).toContain('Menunggu');
  });

  it('renders "Edit Biodata" button for role petugas/admin', () => {
    currentUserSignal.set({ id: 1, nama: 'Staff Petugas', roles: ['petugas'] });
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Edit Biodata');
  });

  it('hides "Edit Biodata" button for role dokter', () => {
    currentUserSignal.set({ id: 2, nama: 'Dr. Dokter', roles: ['dokter'] });
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).not.toContain('Edit Biodata');
  });

  it('renders success toast when history.state contains successMessage', () => {
    component.successMessage.set('Biodata pasien berhasil diperbarui.');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Biodata pasien berhasil diperbarui.');
  });
});
