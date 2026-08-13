import { TestBed } from '@angular/core/testing';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { environment } from '../../../environments/environment';
import { RealtimeService } from './realtime.service';

class MockWebSocket {
  static instances: MockWebSocket[] = [];

  url: string;
  readyState: number = WebSocket.CONNECTING;

  onopen: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  close(): void {
    this.readyState = WebSocket.CLOSED;
    if (this.onclose) {
      this.onclose(new CloseEvent('close'));
    }
  }

  triggerOpen(): void {
    this.readyState = WebSocket.OPEN;
    if (this.onopen) {
      this.onopen(new Event('open'));
    }
  }

  triggerMessage(data: unknown): void {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', { data: JSON.stringify(data) }));
    }
  }

  triggerUnexpectedClose(): void {
    this.readyState = WebSocket.CLOSED;
    if (this.onclose) {
      this.onclose(new CloseEvent('close'));
    }
  }
}

describe('RealtimeService', () => {
  let service: RealtimeService;
  let originalWebSocket: typeof WebSocket;

  beforeEach(() => {
    MockWebSocket.instances = [];
    originalWebSocket = globalThis.WebSocket;
    (globalThis as unknown as { WebSocket: unknown }).WebSocket = MockWebSocket;

    TestBed.configureTestingModule({
      providers: [RealtimeService],
    });
    service = TestBed.inject(RealtimeService);
  });

  afterEach(() => {
    service.disconnect();
    (globalThis as unknown as { WebSocket: unknown }).WebSocket = originalWebSocket;
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it('should be created with initial disconnected status', () => {
    expect(service.connectionStatus()).toBe('disconnected');
    expect(service.lastUpdateAt()).toBeNull();
  });

  it('should connect to WebSocket URL using default klinikId when no options provided', () => {
    service.connect();
    expect(MockWebSocket.instances.length).toBe(1);
    const socket = MockWebSocket.instances[0];
    expect(socket.url).toContain(`/ws?klinikId=${environment.defaultKlinikId}`);
    expect(socket.url).not.toContain('displayToken=');
    expect(service.connectionStatus()).toBe('connecting');
  });

  it('should connect to WebSocket URL with custom klinikId and displayToken when provided', () => {
    service.connect({ klinikId: 99, displayToken: 'my-display-token' });
    expect(MockWebSocket.instances.length).toBe(1);
    const socket = MockWebSocket.instances[0];
    expect(socket.url).toContain('/ws?klinikId=99&displayToken=my-display-token');
  });

  it('should set connectionStatus to connected when WebSocket opens', () => {
    service.connect();
    const socket = MockWebSocket.instances[0];
    socket.triggerOpen();
    expect(service.connectionStatus()).toBe('connected');
  });

  it('should update lastUpdateAt when receiving queue_updated message', () => {
    const startTime = Date.now();
    service.connect();
    const socket = MockWebSocket.instances[0];
    socket.triggerOpen();

    socket.triggerMessage({ type: 'queue_updated' });

    expect(service.lastUpdateAt()).not.toBeNull();
    expect(service.lastUpdateAt()!).toBeGreaterThanOrEqual(startTime);
  });

  it('should not update lastUpdateAt when receiving other message types', () => {
    service.connect();
    const socket = MockWebSocket.instances[0];
    socket.triggerOpen();

    socket.triggerMessage({ type: 'other_event' });
    expect(service.lastUpdateAt()).toBeNull();
  });

  it('should handle unexpected disconnect and trigger reconnect with exponential backoff', () => {
    vi.useFakeTimers();

    service.connect();
    const socket1 = MockWebSocket.instances[0];
    socket1.triggerOpen();
    expect(service.connectionStatus()).toBe('connected');

    socket1.triggerUnexpectedClose();
    expect(service.connectionStatus()).toBe('reconnecting');
    expect(MockWebSocket.instances.length).toBe(1);

    vi.advanceTimersByTime(1500);

    expect(MockWebSocket.instances.length).toBe(2);
    const socket2 = MockWebSocket.instances[1];
    socket2.triggerOpen();
    expect(service.connectionStatus()).toBe('connected');
  });

  it('should reset backoff counter after a successful reconnect', () => {
    vi.useFakeTimers();

    service.connect();
    let socket = MockWebSocket.instances[0];
    socket.triggerOpen();

    socket.triggerUnexpectedClose();
    vi.advanceTimersByTime(1500);
    expect(MockWebSocket.instances.length).toBe(2);

    socket = MockWebSocket.instances[1];
    socket.triggerOpen();

    socket.triggerUnexpectedClose();
    expect(service.connectionStatus()).toBe('reconnecting');

    vi.advanceTimersByTime(1500);
    expect(MockWebSocket.instances.length).toBe(3);
  });

  it('should not trigger reconnect when disconnect() is called manually', () => {
    vi.useFakeTimers();

    service.connect();
    const socket = MockWebSocket.instances[0];
    socket.triggerOpen();

    service.disconnect();
    expect(service.connectionStatus()).toBe('disconnected');

    vi.advanceTimersByTime(10000);
    expect(MockWebSocket.instances.length).toBe(1);
  });
});
