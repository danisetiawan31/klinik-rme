import { ComponentFixture, TestBed } from '@angular/core/testing';
import { toast } from '@spartan-ng/brain/sonner';
import { vi } from 'vitest';
import { ToastComponent } from './toast.component';

describe('ToastComponent', () => {
  let fixture: ComponentFixture<ToastComponent>;

  beforeEach(async () => {
    vi.spyOn(toast, 'error').mockImplementation(() => '1' as any);
    vi.spyOn(toast, 'success').mockImplementation(() => '2' as any);
    vi.spyOn(toast, 'info').mockImplementation(() => '3' as any);

    await TestBed.configureTestingModule({
      imports: [ToastComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(ToastComponent);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should not trigger toast when message is empty', () => {
    fixture.componentRef.setInput('message', '');
    fixture.detectChanges();

    expect(toast.error).not.toHaveBeenCalled();
    expect(toast.success).not.toHaveBeenCalled();
    expect(toast.info).not.toHaveBeenCalled();
  });

  it('should trigger Spartan toast.error for error type', () => {
    fixture.componentRef.setInput('message', 'Email atau password salah');
    fixture.componentRef.setInput('type', 'error');
    fixture.detectChanges();

    expect(toast.error).toHaveBeenCalledWith(
      'Email atau password salah',
      expect.objectContaining({ onDismiss: expect.any(Function) })
    );
  });

  it('should trigger Spartan toast.success for success type', () => {
    fixture.componentRef.setInput('message', 'Data berhasil disimpan');
    fixture.componentRef.setInput('type', 'success');
    fixture.detectChanges();

    expect(toast.success).toHaveBeenCalledWith(
      'Data berhasil disimpan',
      expect.objectContaining({ onDismiss: expect.any(Function) })
    );
  });
});
