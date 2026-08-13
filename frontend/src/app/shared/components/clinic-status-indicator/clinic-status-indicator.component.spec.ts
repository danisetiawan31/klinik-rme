import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { KlinikService } from '../../../core/klinik/klinik.service';
import { KlinikResponse } from '../../../core/klinik/klinik.types';
import { ClinicStatusIndicatorComponent } from './clinic-status-indicator.component';

describe('ClinicStatusIndicatorComponent', () => {
  let component: ClinicStatusIndicatorComponent;
  let fixture: ComponentFixture<ClinicStatusIndicatorComponent>;
  let klinikInfoSignal: WritableSignal<KlinikResponse | null>;
  let klinikServiceSpy: {
    klinikInfo: WritableSignal<KlinikResponse | null>;
    fetchKlinikInfo: ReturnType<typeof vi.fn>;
    isKlinikBuka: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    klinikInfoSignal = signal<KlinikResponse | null>(null);
    klinikServiceSpy = {
      klinikInfo: klinikInfoSignal,
      fetchKlinikInfo: vi.fn().mockReturnValue(of(null)),
      isKlinikBuka: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [ClinicStatusIndicatorComponent],
      providers: [{ provide: KlinikService, useValue: klinikServiceSpy }],
    }).compileComponents();

    fixture = TestBed.createComponent(ClinicStatusIndicatorComponent);
    component = fixture.componentInstance;
  });

  it('should render Klinik Buka badge with semantic token --color-accent and NO pulse or emerald hardcode when open', () => {
    klinikInfoSignal.set({ id: 1, nama: 'Klinik', isBuka: true });
    klinikServiceSpy.isKlinikBuka.mockReturnValue(true);

    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const badgeText = compiled.textContent || '';
    expect(badgeText).toContain('Klinik Buka');

    const badgeEl = compiled.querySelector('span[hlmBadge]') as HTMLElement;
    expect(badgeEl).toBeTruthy();
    expect(badgeEl.className).toContain('text-[var(--color-accent)]');
    expect(badgeEl.className).not.toContain('bg-emerald-500');
    expect(badgeEl.className).not.toContain('border-emerald-200');

    const dotEl = compiled.querySelector('span.rounded-full') as HTMLElement;
    expect(dotEl).toBeTruthy();
    expect(dotEl.className).toContain('bg-[var(--color-accent)]');
    expect(dotEl.className).not.toContain('animate-pulse');
  });

  it('should render Klinik Tutup badge with --color-muted-foreground and NO red/destructive when closed', () => {
    klinikInfoSignal.set({ id: 1, nama: 'Klinik', isBuka: false });
    klinikServiceSpy.isKlinikBuka.mockReturnValue(false);

    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const badgeText = compiled.textContent || '';
    expect(badgeText).toContain('Klinik Tutup');

    const badgeEl = compiled.querySelector('span[hlmBadge]') as HTMLElement;
    expect(badgeEl).toBeTruthy();
    expect(badgeEl.className).toContain('text-[var(--color-muted-foreground)]');
    expect(badgeEl.className).toContain('bg-[var(--color-muted)]');
    expect(badgeEl.className).not.toContain('bg-red-');
    expect(badgeEl.className).not.toContain('text-destructive');
    expect(badgeEl.className).not.toContain('bg-destructive');

    const dotEl = compiled.querySelector('span.rounded-full') as HTMLElement;
    expect(dotEl).toBeTruthy();
    expect(dotEl.className).toContain('bg-[var(--color-muted-foreground)]');
  });
});
