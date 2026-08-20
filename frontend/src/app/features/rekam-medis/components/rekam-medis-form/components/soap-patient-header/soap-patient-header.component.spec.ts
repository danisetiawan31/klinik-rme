import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SoapPatientHeaderComponent } from './soap-patient-header.component';
import { Pasien } from '../../../../../pasien/pasien.types';
import { KunjunganDetail } from '../../../../../antrian/antrian.types';
import { RiwayatRekamMedisItem } from '../../../../rekam-medis.types';

describe('SoapPatientHeaderComponent', () => {
  let component: SoapPatientHeaderComponent;
  let fixture: ComponentFixture<SoapPatientHeaderComponent>;

  const mockPasien: Pasien = {
    id: 1,
    nik: '3201123456789012',
    nama: 'Budi Santoso',
    tanggalLahir: '1990-01-01',
    jenisKelamin: 'L',
    alamat: 'Jl. Melati No. 5',
    noTelp: '08123456789',
    consent: true,
    version: 1,
    riwayatKunjunganRingkas: [],
  };

  const mockKunjungan: KunjunganDetail = {
    id: 10,
    nomorAntrian: 5,
    pasienId: 1,
    status: 'dipanggil',
    isPriority: true,
  };

  const mockRiwayat: RiwayatRekamMedisItem[] = [
    {
      kunjunganId: 8,
      tanggal: '2026-08-10T09:00:00Z',
      rekamMedis: {
        id: 100,
        keluhan: 'Batuk kering',
        hasilPemeriksaan: 'Faring normal',
        diagnosis: [{ id: 1, kodeIcd: 'R05', deskripsi: 'Cough' }],
        tindakan: [{ id: 1, jenis: 'resep', deskripsi: 'OBH Sirup' }],
        createdAt: '2026-08-10T09:30:00Z',
        isAddendum: false,
      },
    },
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SoapPatientHeaderComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(SoapPatientHeaderComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('pasien', mockPasien);
    fixture.componentRef.setInput('kunjungan', mockKunjungan);
    fixture.componentRef.setInput('riwayatList', mockRiwayat);
    fixture.detectChanges();
  });

  it('should render patient identity details and queue badge correctly', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Budi Santoso');
    expect(compiled.textContent).toContain('#005');
    expect(compiled.textContent).toContain('Laki-laki');
    expect(compiled.querySelector('app-priority-badge')).toBeTruthy();
  });

  it('should toggle history drawer when history button is clicked', () => {
    expect(component.showHistory()).toBe(false);

    component.toggleHistory();
    fixture.detectChanges();
    expect(component.showHistory()).toBe(true);

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Histori Pemeriksaan');
    expect(compiled.textContent).toContain('Batuk kering');

    component.toggleHistory();
    fixture.detectChanges();
    expect(component.showHistory()).toBe(false);
  });
});
