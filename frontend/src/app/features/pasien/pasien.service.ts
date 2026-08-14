import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import {
  CreatePasienRequest,
  Pasien,
  PasienSearchItem,
} from './pasien.types';

@Injectable({ providedIn: 'root' })
export class PasienService {
  private http = inject(HttpClient);
  private base = `${environment.apiUrl}/pasien`;

  create(payload: CreatePasienRequest): Observable<Pasien> {
    return this.http.post<Pasien>(this.base, payload);
  }

  /**
   * Dipakai untuk pre-submission NIK duplicate check (Tahap 1).
   * Pagination tidak diimplementasikan di sini — itu scope Tahap 2.
   */
  searchByNik(nik: string): Observable<PasienSearchItem[]> {
    return this.http.get<PasienSearchItem[]>(`${this.base}/search`, {
      params: { nik },
    });
  }
}
