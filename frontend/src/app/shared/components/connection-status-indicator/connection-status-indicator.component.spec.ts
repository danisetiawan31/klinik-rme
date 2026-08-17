import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { ConnectionStatusIndicatorComponent } from './connection-status-indicator.component';
import { ConnectionStatus, RealtimeService } from '../../../core/realtime/realtime.service';

describe('ConnectionStatusIndicatorComponent', () => {
  let component: ConnectionStatusIndicatorComponent;
  let fixture: ComponentFixture<ConnectionStatusIndicatorComponent>;
  let mockStatusSignal: ReturnType<typeof signal<ConnectionStatus>>;

  beforeEach(async () => {
    mockStatusSignal = signal<ConnectionStatus>('disconnected');

    const mockRealtimeService = {
      connectionStatus: mockStatusSignal,
    };

    await TestBed.configureTestingModule({
      imports: [ConnectionStatusIndicatorComponent],
      providers: [
        { provide: RealtimeService, useValue: mockRealtimeService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ConnectionStatusIndicatorComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create the component', () => {
    expect(component).toBeTruthy();
  });

  it('should render "Offline" when status is disconnected', () => {
    mockStatusSignal.set('disconnected');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Offline');
  });

  it('should render "Live" with ping indicator when status is connected', () => {
    mockStatusSignal.set('connected');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Live');
    expect(compiled.querySelector('.animate-ping')).toBeTruthy();
  });

  it('should render "Menyambung ulang…" when status is reconnecting', () => {
    mockStatusSignal.set('reconnecting');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Menyambung ulang…');
  });

  it('should respect manual status input override', () => {
    mockStatusSignal.set('disconnected');
    fixture.componentRef.setInput('status', 'connected');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Live');
  });
});
