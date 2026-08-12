import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ToastComponent } from './toast.component';

describe('ToastComponent', () => {
  let fixture: ComponentFixture<ToastComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ToastComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(ToastComponent);
  });

  it('should not render anything when message is empty', () => {
    fixture.componentRef.setInput('message', '');
    fixture.detectChanges();

    expect(fixture.nativeElement.children.length).toBe(0);
  });

  it('should render top-center toast with role="alert" and aria-live="assertive" for error type', () => {
    fixture.componentRef.setInput('message', 'Email atau password salah');
    fixture.componentRef.setInput('type', 'error');
    fixture.detectChanges();

    const toastEl = fixture.nativeElement.querySelector('[role="alert"]');
    expect(toastEl).toBeTruthy();
    expect(toastEl.getAttribute('aria-live')).toBe('assertive');
    expect(toastEl.textContent).toContain('Email atau password salah');
  });

  it('should emit dismiss event when X button is clicked', () => {
    fixture.componentRef.setInput('message', 'Test Error');
    fixture.detectChanges();

    let dismissed = false;
    fixture.componentInstance.dismiss.subscribe(() => {
      dismissed = true;
    });

    const btn = fixture.nativeElement.querySelector('button[aria-label="Tutup notifikasi"]') as HTMLButtonElement;
    expect(btn).toBeTruthy();
    btn.click();

    expect(dismissed).toBe(true);
  });
});
