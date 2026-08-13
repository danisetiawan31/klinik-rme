import { Injectable, signal } from '@angular/core';
import { environment } from '../../../environments/environment';

export type ConnectionStatus = 'connecting' | 'connected' | 'reconnecting' | 'disconnected';

export interface RealtimeConnectOptions {
  klinikId?: number;
  displayToken?: string;
}

@Injectable({
  providedIn: 'root',
})
export class RealtimeService {
  readonly connectionStatus = signal<ConnectionStatus>('disconnected');
  readonly lastUpdateAt = signal<number | null>(null);

  private socket: WebSocket | null = null;
  private isManualDisconnect = false;
  private currentOptions: RealtimeConnectOptions = {};

  // Backoff constants & state
  private currentDelay = 1000;
  private readonly maxDelay = 30000;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  connect(options?: RealtimeConnectOptions): void {
    if (options) {
      this.currentOptions = options;
    }

    this.isManualDisconnect = false;
    this.clearReconnectTimer();

    if (this.socket) {
      this.socket.onopen = null;
      this.socket.onmessage = null;
      this.socket.onerror = null;
      this.socket.onclose = null;
      this.socket.close();
      this.socket = null;
    }

    if (this.connectionStatus() !== 'reconnecting') {
      this.connectionStatus.set('connecting');
    }

    const klinikId = this.currentOptions.klinikId ?? environment.defaultKlinikId;
    const protocol = typeof window !== 'undefined' && window.location?.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = typeof window !== 'undefined' && window.location?.host ? window.location.host : 'localhost:8080';

    let url = `${protocol}//${host}/ws?klinikId=${klinikId}`;
    if (this.currentOptions.displayToken) {
      url += `&displayToken=${encodeURIComponent(this.currentOptions.displayToken)}`;
    }

    try {
      this.socket = new WebSocket(url);
    } catch {
      this.handleDisconnect();
      return;
    }

    this.socket.onopen = () => {
      this.connectionStatus.set('connected');
      this.currentDelay = 1000;
    };

    this.socket.onmessage = (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data);
        if (data && data.type === 'queue_updated') {
          this.lastUpdateAt.set(Date.now());
        }
      } catch {
        // Ignore non-JSON messages
      }
    };

    this.socket.onerror = () => {
      // Error callback - cleanup handled on close
    };

    this.socket.onclose = () => {
      this.socket = null;
      if (!this.isManualDisconnect) {
        this.handleDisconnect();
      }
    };
  }

  disconnect(): void {
    this.isManualDisconnect = true;
    this.clearReconnectTimer();
    this.connectionStatus.set('disconnected');

    if (this.socket) {
      this.socket.onopen = null;
      this.socket.onmessage = null;
      this.socket.onerror = null;
      this.socket.onclose = null;
      this.socket.close();
      this.socket = null;
    }
  }

  private handleDisconnect(): void {
    if (this.isManualDisconnect) {
      return;
    }

    this.connectionStatus.set('reconnecting');
    this.scheduleReconnect();
  }

  private scheduleReconnect(): void {
    this.clearReconnectTimer();

    const jitterMultiplier = 0.8 + Math.random() * 0.4;
    const actualDelay = Math.round(this.currentDelay * jitterMultiplier);

    this.currentDelay = Math.min(this.currentDelay * 2, this.maxDelay);

    this.reconnectTimer = setTimeout(() => {
      if (!this.isManualDisconnect) {
        this.connect();
      }
    }, actualDelay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}
