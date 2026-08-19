import { HttpClient } from '@angular/common/http';
import { inject, Injectable, signal } from '@angular/core';
import { catchError, Observable, of, tap } from 'rxjs';
import { environment } from '../../../environments/environment';
import { getJakartaTimeString } from '../utils/date.utils';
import { KlinikResponse } from './klinik.types';

@Injectable({
  providedIn: 'root',
})
export class KlinikService {
  private http = inject(HttpClient);

  readonly klinikInfo = signal<KlinikResponse | null>(null);
  readonly isLoading = signal<boolean>(false);

  /**
   * Helper method to compute whether clinic is currently open (isBuka)
   * anchored strictly to Asia/Jakarta (WIB) timezone per AGENTS.md §7 & api-contract.md.
   */
  isKlinikBuka(info: KlinikResponse | null = this.klinikInfo()): boolean {
    if (!info) {
      return true; // Default fallback open if data not yet loaded
    }

    if (typeof info.isBuka === 'boolean') {
      return info.isBuka;
    }

    if (!info.jamBuka || !info.jamTutup) {
      return true;
    }

    try {
      const currentJakartaTime = getJakartaTimeString(new Date());
      return currentJakartaTime >= info.jamBuka && currentJakartaTime < info.jamTutup;
    } catch {
      return true;
    }
  }

  fetchKlinikInfo(klinikId: number = environment.defaultKlinikId, displayToken?: string): Observable<KlinikResponse | null> {
    this.isLoading.set(true);
    let headers: Record<string, string> = {};
    if (displayToken) {
      headers['X-Display-Token'] = displayToken;
    }
    return this.http.get<KlinikResponse>(`${environment.apiUrl}/klinik/${klinikId}`, { headers }).pipe(
      tap((info) => {
        this.klinikInfo.set(info);
        this.isLoading.set(false);
      }),
      catchError(() => {
        this.isLoading.set(false);
        return of(null);
      })
    );
  }
}
