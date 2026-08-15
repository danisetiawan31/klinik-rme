import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, provideRouter, Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthService } from '../../../../core/auth/auth.service';
import { KlinikService } from '../../../../core/klinik/klinik.service';
import { AntrianService } from '../../../antrian/antrian.service';
import { PasienService } from '../../pasien.service';
import { Pasien } from '../../pasien.types';
import { PasienDetailComponent } from './pasien-detail.component';

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

describe('PasienDetailComponent (with Tahap 3 Queue Registration)', () => {
  let component: PasienDetailComponent;
  let fixture: ComponentFixture<PasienDetailComponent>;
  let router: Router;
  let mockPasienService: any;
  let mockAntrianService: any;
  let mockKlinikService: any;
  let currentUserSignal = signal<{ id: number; nama: string; roles: string[] } | null>({
    id: 1,
    nama: 'Staff Petugas',
    roles: ['petugas'],
  });

  beforeEach(async () => {
    vi.clearAllMocks();
    currentUserSignal.set({ id: 1, nama: 'Staff Petugas', roles: ['petugas'] });

    mockPasienService = {
      getById: vi.fn().mockReturnValue(of(mockPasien)),
    };

    mockAntrianService = {
      create: vi.fn(),
    };

    mockKlinikService = {
      klinikInfo: signal({ id: 1, nama: 'Klinik Sehat', isBuka: true }),
      isKlinikBuka: vi.fn().mockReturnValue(true),
      fetchKlinikInfo: vi.fn().mockReturnValue(of({ id: 1, nama: 'Klinik Sehat', isBuka: true })),
    };

    await TestBed.configureTestingModule({
      imports: [PasienDetailComponent],
      providers: [
        provideRouter([]),
        { provide: PasienService, useValue: mockPasienService },
        { provide: AntrianService, useValue: mockAntrianService },
        { provide: KlinikService, useValue: mockKlinikService },
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

    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigate');

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

  it('renders "Edit Biodata" and "Daftarkan ke Antrian" buttons for role petugas/admin', () => {
    currentUserSignal.set({ id: 1, nama: 'Staff Petugas', roles: ['petugas'] });
    fixture.detectChanges();

    const textContent = fixture.nativeElement.textContent;
    expect(textContent).toContain('Edit Biodata');
    expect(textContent).toContain('Daftarkan ke Antrian');
  });

  it('hides "Edit Biodata" and "Daftarkan ke Antrian" buttons for role dokter', () => {
    currentUserSignal.set({ id: 2, nama: 'Dr. Dokter', roles: ['dokter'] });
    fixture.detectChanges();

    const textContent = fixture.nativeElement.textContent;
    expect(textContent).not.toContain('Edit Biodata');
    expect(textContent).not.toContain('Daftarkan ke Antrian');
  });

  it('disables "Daftarkan ke Antrian" button when isKlinikBuka() is false', () => {
    mockKlinikService.isKlinikBuka.mockReturnValue(false);
    mockKlinikService.klinikInfo.set({ id: 1, nama: 'Klinik Sehat', isBuka: false });
    fixture.detectChanges();

    const buttons = fixture.nativeElement.querySelectorAll('button');
    const daftarBtn = Array.from(buttons).find((b: any) =>
      b.textContent.includes('Daftarkan ke Antrian')
    ) as HTMLButtonElement;

    expect(daftarBtn).toBeTruthy();
    expect(daftarBtn.disabled).toBe(true);
    expect(daftarBtn.title).toContain('Pendaftaran ditutup');
  });

  it('opens and closes registration modal', () => {
    component.openDaftarModal();
    fixture.detectChanges();

    expect(component.showDaftarModal()).toBe(true);
    // Spartan Dialog renders via CDK overlay portal into document.body (outside nativeElement)
    expect(document.body.textContent).toContain('Daftarkan ke Antrian Hari Ini');

    component.closeDaftarModal();
    fixture.detectChanges();

    expect(component.showDaftarModal()).toBe(false);
  });

  it('requires priorityReason when isPriority is true and blocks submit', () => {
    component.openDaftarModal();
    component.isPriority.set(true);
    component.priorityReason.set('   ');
    fixture.detectChanges();

    component.submitDaftarAntrian();
    fixture.detectChanges();

    expect(mockAntrianService.create).not.toHaveBeenCalled();
    expect(component.formError()).toContain('Alasan prioritas wajib diisi');
    expect(component.showDaftarModal()).toBe(true);
  });

  it('successfully submits queue registration, shows success toast with nomorAntrian, and does NOT navigate', () => {
    mockAntrianService.create.mockReturnValue(
      of({ id: 99, nomorAntrian: 8, status: 'menunggu', tanggalKunjungan: '2026-08-14' })
    );
    mockPasienService.getById.mockClear();

    component.openDaftarModal();
    component.isPriority.set(true);
    component.priorityReason.set('Lansia dengan kursi roda');
    fixture.detectChanges();

    component.submitDaftarAntrian();
    fixture.detectChanges();

    expect(mockAntrianService.create).toHaveBeenCalledWith({
      pasienId: 42,
      isPriority: true,
      priorityReason: 'Lansia dengan kursi roda',
    });
    expect(component.showDaftarModal()).toBe(false);
    expect(component.successMessage()).toContain('Nomor Antrian #8');
    // Refetches detail to update riwayat list
    expect(mockPasienService.getById).toHaveBeenCalledWith(42);
    // Does NOT navigate away
    expect(router.navigate).not.toHaveBeenCalled();
  });

  it('handles race condition error (e.g. 400 KLINIK_TUTUP) by showing toast error and KEEPING modal open', () => {
    mockAntrianService.create.mockReturnValue(
      throwError(() => ({
        error: { code: 'KLINIK_TUTUP', message: 'Pendaftaran antrian sudah ditutup untuk hari ini' },
      }))
    );

    component.openDaftarModal();
    fixture.detectChanges();

    component.submitDaftarAntrian();
    fixture.detectChanges();

    expect(component.errorMessage()).toBe('Pendaftaran antrian sudah ditutup untuk hari ini.');
    expect(component.isSubmitting()).toBe(false);
    // Modal MUST stay open on error
    expect(component.showDaftarModal()).toBe(true);
  });
});
