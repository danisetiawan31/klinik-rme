import { ComponentFixture, TestBed } from '@angular/core/testing';
import { StatusBadgeComponent } from './status-badge.component';

describe('StatusBadgeComponent', () => {
  let component: StatusBadgeComponent;
  let fixture: ComponentFixture<StatusBadgeComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StatusBadgeComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(StatusBadgeComponent);
    component = fixture.componentInstance;
  });

  it('should render "Menunggu" variant with clock icon correctly', () => {
    fixture.componentRef.setInput('status', 'menunggu');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('Menunggu');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.className).toContain('text-amber-700');
    expect(badge.className).toContain('border-amber-500/30');
    expect(badge.className).toContain('rounded-md');
  });

  it('should render "Dipanggil" variant with megaphone icon and pulse glow correctly', () => {
    fixture.componentRef.setInput('status', 'dipanggil');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('Dipanggil');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.className).toContain('bg-primary');
    expect(badge.className).toContain('text-primary-foreground');
    expect(badge.className).toContain('animate-pulse');
  });

  it('should render "Selesai" variant with check icon correctly', () => {
    fixture.componentRef.setInput('status', 'selesai');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('Selesai');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.className).toContain('text-emerald-700');
    expect(badge.className).toContain('border-emerald-500/30');
  });

  it('should render "Tidak Hadir" variant with user-x icon correctly', () => {
    fixture.componentRef.setInput('status', 'tidak_hadir');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('Tidak Hadir');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.className).toContain('text-muted-foreground');
  });

  it('should render compact xs icon-only variant when size="xs"', () => {
    fixture.componentRef.setInput('status', 'menunggu');
    fixture.componentRef.setInput('size', 'xs');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.className).toContain('size-6');
  });
});
