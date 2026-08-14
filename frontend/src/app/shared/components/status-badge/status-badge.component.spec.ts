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

  it('should render "Menunggu" variant correctly', () => {
    fixture.componentRef.setInput('status', 'menunggu');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('Menunggu');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.className).toContain('text-warning-foreground');
  });

  it('should render "Dipanggil" variant correctly', () => {
    fixture.componentRef.setInput('status', 'dipanggil');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('Dipanggil');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.className).toContain('bg-primary');
    expect(badge.className).toContain('text-primary-foreground');
  });

  it('should render "Selesai" variant correctly', () => {
    fixture.componentRef.setInput('status', 'selesai');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('Selesai');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.className).toContain('bg-accent');
    expect(badge.className).toContain('text-accent-foreground');
  });

  it('should render "Tidak Hadir" variant correctly', () => {
    fixture.componentRef.setInput('status', 'tidak_hadir');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('Tidak Hadir');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.className).toContain('text-muted-foreground');
  });
});
