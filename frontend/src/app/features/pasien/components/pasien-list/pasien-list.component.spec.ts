import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { of } from 'rxjs';
import { PasienListComponent } from './pasien-list.component';
import { PasienService } from '../../pasien.service';

const mockPasienService = {
  search: vi.fn(),
};

describe('PasienListComponent', () => {
  let component: PasienListComponent;
  let fixture: ComponentFixture<PasienListComponent>;
  let router: Router;

  beforeEach(async () => {
    vi.clearAllMocks();
    mockPasienService.search.mockReturnValue(
      of({
        items: [
          { id: 1, nik: '1234567890123456', nama: 'Budi Santoso', tanggalLahir: '1990-01-01' },
          { id: 2, nik: null, nama: 'Siti Aminah', tanggalLahir: '1995-05-05' },
        ],
        totalCount: 2,
      })
    );

    await TestBed.configureTestingModule({
      imports: [PasienListComponent],
      providers: [
        provideRouter([]),
        { provide: PasienService, useValue: mockPasienService },
      ],
    }).compileComponents();

    router = TestBed.inject(Router);
    fixture = TestBed.createComponent(PasienListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('fetches initial data on init', () => {
    expect(mockPasienService.search).toHaveBeenCalledWith({
      nama: undefined,
      nik: undefined,
      page: 1,
      limit: 10,
    });
    expect(component.items().length).toBe(2);
  });

  // ── Nama search trigger (debounce 300ms) ──────────────────────────────────

  it('nama search does NOT trigger API call immediately (wait 300ms debounce)', () => {
    vi.useFakeTimers();
    mockPasienService.search.mockClear();

    component.searchForm.controls.nama.setValue('Ahmad');
    fixture.detectChanges();

    // Not called immediately
    expect(mockPasienService.search).not.toHaveBeenCalled();

    // Advance 290ms
    vi.advanceTimersByTime(290);
    expect(mockPasienService.search).not.toHaveBeenCalled();

    // Advance past 300ms
    vi.advanceTimersByTime(20);
    expect(mockPasienService.search).toHaveBeenCalledWith({
      nama: 'Ahmad',
      nik: undefined,
      page: 1,
      limit: 10,
    });

    vi.useRealTimers();
  });

  // ── NIK search trigger (exact 16 digits) ──────────────────────────────────

  it('NIK search does NOT trigger when NIK is less than 16 digits', () => {
    mockPasienService.search.mockClear();

    component.searchForm.controls.nik.setValue('123456789012345'); // 15 digits
    fixture.detectChanges();

    expect(mockPasienService.search).not.toHaveBeenCalled();
  });

  it('NIK search triggers immediately when NIK hits 16 digits', () => {
    mockPasienService.search.mockClear();

    component.searchForm.controls.nik.setValue('1234567890123456'); // 16 digits
    fixture.detectChanges();

    expect(mockPasienService.search).toHaveBeenCalledWith({
      nama: undefined,
      nik: '1234567890123456',
      page: 1,
      limit: 10,
    });
  });

  // ── Page reset on query change ───────────────────────────────────────────

  it('resets page to 1 when search query (nama) changes from page 3', () => {
    vi.useFakeTimers();
    component.page.set(3);
    mockPasienService.search.mockClear();

    component.searchForm.controls.nama.setValue('Budi');
    vi.advanceTimersByTime(300);

    expect(component.page()).toBe(1);
    expect(mockPasienService.search).toHaveBeenCalledWith({
      nama: 'Budi',
      nik: undefined,
      page: 1,
      limit: 10,
    });

    vi.useRealTimers();
  });

  it('resets page to 1 when search query (nik 16 digit) changes from page 2', () => {
    component.page.set(2);
    mockPasienService.search.mockClear();

    component.searchForm.controls.nik.setValue('9876543210987654');

    expect(component.page()).toBe(1);
    expect(mockPasienService.search).toHaveBeenCalledWith({
      nama: undefined,
      nik: '9876543210987654',
      page: 1,
      limit: 10,
    });
  });

  // ── Navigation ────────────────────────────────────────────────────────────

  it('navigates to /pasien/:id on row click', () => {
    const navigateSpy = vi.spyOn(router, 'navigate');
    component.onSelectPasien(42);
    expect(navigateSpy).toHaveBeenCalledWith(['/pasien', 42]);
  });
});
