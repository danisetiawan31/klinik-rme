import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RevealOnceSecretComponent } from './reveal-once-secret.component';

describe('RevealOnceSecretComponent', () => {
  let component: RevealOnceSecretComponent;
  let fixture: ComponentFixture<RevealOnceSecretComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RevealOnceSecretComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(RevealOnceSecretComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('secretValue', 'http://localhost:4200/set-password?token=secret123');
    fixture.detectChanges();
  });

  it('should create and render secret value', () => {
    expect(component).toBeTruthy();
    expect(component.secretValue()).toBe('http://localhost:4200/set-password?token=secret123');
    const inputEl = fixture.nativeElement.querySelector('input');
    expect(inputEl.value).toBe('http://localhost:4200/set-password?token=secret123');
  });

  it('should copy secret to clipboard when copy button clicked', async () => {
    const writeTextSpy = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: {
        writeText: writeTextSpy,
      },
    });

    component.copyToClipboard();
    expect(writeTextSpy).toHaveBeenCalledWith('http://localhost:4200/set-password?token=secret123');
  });

  it('should emit closed event when close button clicked', () => {
    const emitSpy = vi.fn();
    component.closed.subscribe(emitSpy);

    component.onClose();
    expect(emitSpy).toHaveBeenCalled();
  });
});
