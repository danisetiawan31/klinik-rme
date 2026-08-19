import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthService } from '../../core/auth/auth.service';
import { KlinikService } from '../../core/klinik/klinik.service';
import { RealtimeService } from '../../core/realtime/realtime.service';
import { AntrianDashboardComponent } from './antrian-dashboard.component';
import { AntrianService } from './antrian.service';
import { KunjunganListItem } from './antrian.types';

describe('AntrianDashboardComponent', () => {
  let component: AntrianDashboardComponent;
  let fixture: ComponentFixture<AntrianDashboardComponent>;
  let antrianServiceMock: any;
  let realtimeServiceMock: any;
  let klinikServiceMock: any;
  let authServiceMock: any;

  const mockAntrianData: KunjunganListItem[] = [
    {
      id: 1,
      nomorAntrian: 1,
      status: 'menunggu',
      isPriority: false,
      priorityReason: null,
      skipCount: 0,
      pasienNama: 'Pasien Menunggu',
    },
    {
      id: 2,
      nomorAntrian: 2,
      status: 'dipanggil',
      isPriority: true,
      priorityReason: 'Lansia',
      skipCount: 0,
      pasienNama: 'Pasien Dipanggil',
    },
    {
      id: 3,
      nomorAntrian: 3,
      status: 'selesai',
      isPriority: false,
      priorityReason: null,
      skipCount: 1,
      pasienNama: 'Pasien Selesai',
    },
  ];

  beforeEach(async () => {
    antrianServiceMock = {
      getAntrian: vi.fn().mockReturnValue(of(mockAntrianData)),
      panggilBerikutnya: vi.fn(),
      lewati: vi.fn(),
      tidakHadir: vi.fn(),
    };

    realtimeServiceMock = new RealtimeService();
    vi.spyOn(realtimeServiceMock, 'connect');
    vi.spyOn(realtimeServiceMock, 'disconnect');

    klinikServiceMock = {
      klinikInfo: signal({ id: 1, nama: 'Klinik Sehat', isBuka: true }),
      isKlinikBuka: vi.fn().mockReturnValue(true),
      fetchKlinikInfo: vi.fn().mockReturnValue(of({ id: 1, nama: 'Klinik Sehat', isBuka: true })),
    };

    authServiceMock = {
      currentUser: signal({ id: 1, nama: 'Dokter Budi', roles: ['dokter'] }),
    };

    await TestBed.configureTestingModule({
      imports: [AntrianDashboardComponent],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AntrianService, useValue: antrianServiceMock },
        { provide: RealtimeService, useValue: realtimeServiceMock },
        { provide: KlinikService, useValue: klinikServiceMock },
        { provide: AuthService, useValue: authServiceMock },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(AntrianDashboardComponent);
    component = fixture.componentInstance;
  });

  describe('Tahap 1 Baseline Tests', () => {
    it('should create and call realtimeService.connect() on init', () => {
      fixture.detectChanges();
      expect(component).toBeTruthy();
      expect(realtimeServiceMock.connect).toHaveBeenCalled();
      expect(antrianServiceMock.getAntrian).toHaveBeenCalled();
    });

    it('should correctly sort antrian list: isPriority DESC, skipCount ASC, nomorAntrian ASC', () => {
      fixture.detectChanges();

      const sorted = component.sortedAntrian();
      expect(sorted.length).toBe(3);
      // 1st: Pasien Dipanggil (isPriority: true, skipCount: 0, no: 2)
      expect(sorted[0].id).toBe(2);
      // 2nd: Pasien Menunggu (isPriority: false, skipCount: 0, no: 1)
      expect(sorted[1].id).toBe(1);
      // 3rd: Pasien Selesai (isPriority: false, skipCount: 1, no: 3)
      expect(sorted[2].id).toBe(3);
    });

    it('should compute summary counts accurately', () => {
      fixture.detectChanges();

      expect(component.totalCount()).toBe(3);
      expect(component.menungguCount()).toBe(1);
      expect(component.dipanggilCount()).toBe(1);
      expect(component.selesaiCount()).toBe(1);
    });

    it('should trigger refetch when lastUpdateAt signal changes', () => {
      fixture.detectChanges();
      antrianServiceMock.getAntrian.mockClear();

      realtimeServiceMock.lastUpdateAt.set(Date.now());
      TestBed.flushEffects();

      expect(antrianServiceMock.getAntrian).toHaveBeenCalledTimes(1);
    });

    it('should trigger refetch when connectionStatus transitions to connected', () => {
      fixture.detectChanges();
      antrianServiceMock.getAntrian.mockClear();

      realtimeServiceMock.connectionStatus.set('reconnecting');
      TestBed.flushEffects();
      expect(antrianServiceMock.getAntrian).not.toHaveBeenCalled();

      realtimeServiceMock.connectionStatus.set('connected');
      TestBed.flushEffects();
      expect(antrianServiceMock.getAntrian).toHaveBeenCalledTimes(1);
    });

    it('should call realtimeService.disconnect() when destroyed', () => {
      fixture.detectChanges();
      fixture.destroy();

      expect(realtimeServiceMock.disconnect).toHaveBeenCalled();
    });

    it('should render empty state when antrian list is empty', () => {
      antrianServiceMock.getAntrian.mockReturnValue(of([]));
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      expect(compiled.textContent).toContain('Belum ada antrian hari ini');
    });
  });

  describe('Tahap 2 RBAC & Button Visibility Tests', () => {
    it('should show all doctor action buttons for role dokter', () => {
      authServiceMock.currentUser.set({ id: 1, nama: 'Dr. John', roles: ['dokter'] });
      fixture.detectChanges();

      const el = fixture.nativeElement;
      // Global Panggil button exists
      expect(el.textContent).toContain('Panggil Berikutnya');

      // Table action buttons
      const buttons = el.querySelectorAll('table button');
      const buttonTexts = Array.from(buttons).map((b: any) => b.textContent.trim());

      // Should have "Lewati" on dipanggil row and "Tidak Hadir" on menunggu row
      expect(buttonTexts).toContain('Lewati');
      expect(buttonTexts).toContain('Tidak Hadir');
    });

    it('should show only "Tidak Hadir" for role admin (no Panggil / Lewati)', () => {
      authServiceMock.currentUser.set({ id: 2, nama: 'Admin Super', roles: ['admin'] });
      fixture.detectChanges();

      const el = fixture.nativeElement;
      // No global Panggil button
      expect(el.textContent).not.toContain('Panggil Berikutnya');

      // Table action buttons
      const buttons = el.querySelectorAll('table button');
      const buttonTexts = Array.from(buttons).map((b: any) => b.textContent.trim());

      expect(buttonTexts).not.toContain('Lewati');
      expect(buttonTexts).toContain('Tidak Hadir');
    });

    it('should show NO action buttons for role petugas (view-only)', () => {
      authServiceMock.currentUser.set({ id: 3, nama: 'Petugas Andi', roles: ['petugas'] });
      fixture.detectChanges();

      const el = fixture.nativeElement;
      expect(el.textContent).not.toContain('Panggil Berikutnya');

      // No action column in table
      const headers = el.querySelectorAll('th');
      const headerTexts = Array.from(headers).map((h: any) => h.textContent.trim());
      expect(headerTexts).not.toContain('Aksi');

      const buttons = el.querySelectorAll('table button');
      expect(buttons.length).toBe(0);
    });
  });

  describe('Tahap 2 Doctor & Staff Action Execution Tests', () => {
    it('should handle 204 No Content from panggilBerikutnya with info toast', () => {
      authServiceMock.currentUser.set({ id: 1, nama: 'Dr. John', roles: ['dokter'] });
      antrianServiceMock.panggilBerikutnya.mockReturnValue(of(null));
      fixture.detectChanges();

      component.onPanggilBerikutnya();
      fixture.detectChanges();

      expect(antrianServiceMock.panggilBerikutnya).toHaveBeenCalled();
      expect(component.toastType()).toBe('info');
      expect(component.toastMessage()).toContain('Antrian kosong');
    });

    it('should handle 200 OK from panggilBerikutnya with success toast and refetch list', () => {
      authServiceMock.currentUser.set({ id: 1, nama: 'Dr. John', roles: ['dokter'] });
      antrianServiceMock.panggilBerikutnya.mockReturnValue(
        of({ id: 2, nomorAntrian: 2, pasienNama: 'Siti Rahma', dokterId: 1, dipanggilAt: '2026-08-14T09:00:00Z' })
      );
      fixture.detectChanges();
      antrianServiceMock.getAntrian.mockClear();

      component.onPanggilBerikutnya();
      fixture.detectChanges();

      expect(component.toastType()).toBe('success');
      expect(component.toastMessage()).toContain('Berhasil memanggil antrian #2');
      expect(antrianServiceMock.getAntrian).toHaveBeenCalled();
    });

    it('should handle lewati with success toast and refetch list', () => {
      authServiceMock.currentUser.set({ id: 1, nama: 'Dr. John', roles: ['dokter'] });
      antrianServiceMock.lewati.mockReturnValue(of({ id: 2, status: 'menunggu', skipCount: 1 }));
      fixture.detectChanges();
      antrianServiceMock.getAntrian.mockClear();

      const dipanggilItem = mockAntrianData.find((k) => k.id === 2)!;
      component.onLewati(dipanggilItem);
      fixture.detectChanges();

      expect(antrianServiceMock.lewati).toHaveBeenCalledWith(2);
      expect(component.toastType()).toBe('success');
      expect(component.toastMessage()).toContain('dilewati');
      expect(antrianServiceMock.getAntrian).toHaveBeenCalled();
    });

    it('should refetch antrian list when onLewati encounters 409 conflict error', () => {
      authServiceMock.currentUser.set({ id: 1, nama: 'Dr. John', roles: ['dokter'] });
      antrianServiceMock.lewati.mockReturnValue(
        throwError(() => ({ error: { message: 'Status kunjungan bukan dipanggil' } }))
      );
      fixture.detectChanges();
      antrianServiceMock.getAntrian.mockClear();

      const dipanggilItem = mockAntrianData.find((k) => k.id === 2)!;
      component.onLewati(dipanggilItem);
      fixture.detectChanges();

      expect(component.toastType()).toBe('error');
      expect(component.toastMessage()).toBe('Status kunjungan bukan dipanggil');
      // MUST refetch on 409 error
      expect(antrianServiceMock.getAntrian).toHaveBeenCalled();
    });

    it('should open modal on openConfirmTidakHadir and do nothing on cancel', () => {
      authServiceMock.currentUser.set({ id: 1, nama: 'Dr. John', roles: ['dokter'] });
      fixture.detectChanges();

      const menungguItem = mockAntrianData.find((k) => k.id === 1)!;
      component.openConfirmTidakHadir(menungguItem);
      fixture.detectChanges();

      expect(component.confirmTidakHadirKunjungan()).toEqual(menungguItem);
      // Spartan Dialog renders via CDK overlay portal into document.body (outside nativeElement)
      expect(document.body.textContent).toContain('Konfirmasi Tidak Hadir');

      // Cancel
      component.cancelConfirmTidakHadir();
      fixture.detectChanges();

      expect(component.confirmTidakHadirKunjungan()).toBeNull();
      expect(antrianServiceMock.tidakHadir).not.toHaveBeenCalled();
    });

    it('should execute tidakHadir on modal confirmation, show success toast, and refetch list', () => {
      authServiceMock.currentUser.set({ id: 1, nama: 'Dr. John', roles: ['dokter'] });
      antrianServiceMock.tidakHadir.mockReturnValue(of({ id: 1, status: 'tidak_hadir' }));
      fixture.detectChanges();
      antrianServiceMock.getAntrian.mockClear();

      const menungguItem = mockAntrianData.find((k) => k.id === 1)!;
      component.openConfirmTidakHadir(menungguItem);
      component.executeTidakHadir();
      fixture.detectChanges();

      expect(antrianServiceMock.tidakHadir).toHaveBeenCalledWith(1);
      expect(component.confirmTidakHadirKunjungan()).toBeNull();
      expect(component.toastType()).toBe('success');
      expect(component.toastMessage()).toContain('ditandai tidak hadir');
      expect(antrianServiceMock.getAntrian).toHaveBeenCalled();
    });
  });
});
