import { ComponentFixture, TestBed } from '@angular/core/testing';
import { PriorityBadgeComponent } from './priority-badge.component';

describe('PriorityBadgeComponent', () => {
  let component: PriorityBadgeComponent;
  let fixture: ComponentFixture<PriorityBadgeComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PriorityBadgeComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(PriorityBadgeComponent);
    component = fixture.componentInstance;
  });

  it('should render priority badge with label and title', () => {
    fixture.componentRef.setInput('reason', 'Lansia 75 tahun');
    fixture.detectChanges();

    const el = fixture.nativeElement;
    expect(el.textContent.trim()).toBe('Prioritas');
    const badge = el.querySelector('[hlmBadge]');
    expect(badge.getAttribute('title')).toBe('Pasien Prioritas: Lansia 75 tahun');
    expect(badge.className).toContain('text-foreground');
    expect(badge.className).toContain('border-secondary');
  });

  it('should render fallback title if reason is not provided', () => {
    fixture.componentRef.setInput('reason', null);
    fixture.detectChanges();

    const badge = fixture.nativeElement.querySelector('[hlmBadge]');
    expect(badge.getAttribute('title')).toBe('Pasien Prioritas');
  });
});
