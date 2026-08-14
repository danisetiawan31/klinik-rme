import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { BrnDialogRef } from '@spartan-ng/brain/dialog';
import { of } from 'rxjs';
import { AuthService } from '../../core/auth/auth.service';
import { UserResponse } from '../../core/auth/auth.types';
import { KlinikService } from '../../core/klinik/klinik.service';
import { KlinikResponse } from '../../core/klinik/klinik.types';
import { ShellComponent } from './shell.component';

describe('ShellComponent', () => {
  let component: ShellComponent;
  let fixture: ComponentFixture<ShellComponent>;
  let userSignal: WritableSignal<UserResponse | null>;
  let authServiceSpy: {
    currentUser: WritableSignal<UserResponse | null>;
    logout: ReturnType<typeof vi.fn>;
  };
  let klinikServiceSpy: {
    klinikInfo: WritableSignal<KlinikResponse | null>;
    fetchKlinikInfo: ReturnType<typeof vi.fn>;
    isKlinikBuka: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    userSignal = signal<UserResponse | null>(null);
    authServiceSpy = {
      currentUser: userSignal,
      logout: vi.fn().mockReturnValue(of(undefined)),
    };

    klinikServiceSpy = {
      klinikInfo: signal<KlinikResponse | null>({ id: 1, nama: 'Klinik Test', isBuka: true }),
      fetchKlinikInfo: vi.fn().mockReturnValue(of(null)),
      isKlinikBuka: vi.fn().mockReturnValue(true),
    };

    await TestBed.configureTestingModule({
      imports: [ShellComponent],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: authServiceSpy },
        { provide: KlinikService, useValue: klinikServiceSpy },
        { provide: BrnDialogRef, useValue: { close: vi.fn() } },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ShellComponent);
    component = fixture.componentInstance;
  });

  it('should render navigation links correctly for role petugas and exclude role-restricted links', () => {
    userSignal.set({ id: 1, nama: 'Petugas A', roles: ['petugas'] });
    fixture.detectChanges();

    const routes = component.navItems().map((item) => item.route);

    expect(routes).toContain('/');
    expect(routes).toContain('/pasien');
    expect(routes).toContain('/antrian');
    expect(routes).toContain('/laporan-harian');

    // Negative assertions
    expect(routes).not.toContain('/rekam-medis');
    expect(routes).not.toContain('/admin/users');
    expect(routes).not.toContain('/admin/audit-log');
    expect(routes).not.toContain('/admin/pengaturan');
  });

  it('should render navigation links correctly for role dokter and exclude admin links', () => {
    userSignal.set({ id: 2, nama: 'Dr. Budi', roles: ['dokter'] });
    fixture.detectChanges();

    const routes = component.navItems().map((item) => item.route);

    expect(routes).toContain('/');
    expect(routes).toContain('/antrian');
    expect(routes).toContain('/rekam-medis');
    expect(routes).toContain('/pasien/riwayat');
    expect(routes).toContain('/laporan-harian');

    // Negative assertions
    expect(routes).not.toContain('/admin/users');
    expect(routes).not.toContain('/admin/audit-log');
    expect(routes).not.toContain('/admin/pengaturan');
  });

  it('should render navigation links correctly for role admin and exclude dokter-only links', () => {
    userSignal.set({ id: 3, nama: 'Admin C', roles: ['admin'] });
    fixture.detectChanges();

    const routes = component.navItems().map((item) => item.route);

    expect(routes).toContain('/');
    expect(routes).toContain('/pasien');
    expect(routes).toContain('/antrian');
    expect(routes).toContain('/admin/users');
    expect(routes).toContain('/admin/audit-log');
    expect(routes).toContain('/admin/pengaturan');
    expect(routes).toContain('/laporan-harian');

    // Negative assertion
    expect(routes).not.toContain('/rekam-medis');
  });

  it('should invoke authService.logout when logout is called', () => {
    userSignal.set({ id: 1, nama: 'Staff', roles: ['petugas'] });
    fixture.detectChanges();

    component.logout();

    expect(authServiceSpy.logout).toHaveBeenCalled();
  });

  it('should render user menu link to /profil for Pengaturan Akun when user menu is opened', () => {
    userSignal.set({ id: 1, nama: 'Staff', roles: ['petugas'] });
    fixture.detectChanges();

    const menuTrigger = fixture.nativeElement.querySelector('button[aria-label="Menu Pengguna"]');
    expect(menuTrigger).toBeTruthy();

    menuTrigger.click();
    fixture.detectChanges();

    const profilLink = document.querySelector('a[routerLink="/profil"]');
    expect(profilLink).toBeTruthy();
    expect(profilLink?.textContent).toContain('Pengaturan Akun');
  });
});
