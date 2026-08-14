import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import {
  CreatePasienRequest,
  Pasien,
  PasienSearchItem,
  UpdatePasienRequest,
} from './pasien.types';

export interface PasienSearchParams {
  nik?: string;
  nama?: string;
  page?: number;
  limit?: number;
}

export interface PasienSearchResult {
  items: PasienSearchItem[];
  totalCount: number;
}

@Injectable({ providedIn: 'root' })
export class PasienService {
  private http = inject(HttpClient);
  private base = `${environment.apiUrl}/pasien`;

  create(payload: CreatePasienRequest): Observable<Pasien> {
    return this.http.post<Pasien>(this.base, payload);
  }

  /**
   * Search pasien dengan filter nik/nama + pagination (page/limit).
   * Membaca header X-Total-Count dari response HTTP (`observe: 'response'`).
   */
  search(params: PasienSearchParams): Observable<PasienSearchResult> {
    let httpParams = new HttpParams();
    if (params.nik?.trim()) {
      httpParams = httpParams.set('nik', params.nik.trim());
    }
    if (params.nama?.trim()) {
      httpParams = httpParams.set('nama', params.nama.trim());
    }
    if (params.page !== undefined && params.page > 0) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params.limit !== undefined && params.limit > 0) {
      httpParams = httpParams.set('limit', params.limit.toString());
    }

    return this.http
      .get<PasienSearchItem[]>(`${this.base}/search`, {
        params: httpParams,
        observe: 'response',
      })
      .pipe(
        map((response) => {
          const totalHeader = response.headers.get('X-Total-Count');
          const totalCount = totalHeader ? parseInt(totalHeader, 10) : 0;
          return {
            items: response.body || [],
            totalCount: isNaN(totalCount) ? (response.body?.length || 0) : totalCount,
          };
        })
      );
  }

  /**
   * Helper wrapper untuk Tahap 1 NIK duplicate check pre-submission.
   */
  searchByNik(nik: string): Observable<PasienSearchItem[]> {
    return this.search({ nik }).pipe(map((res) => res.items));
  }

  /**
   * Fetch detail pasien lengkap & riwayat kunjungan ringkas.
   */
  getById(id: number): Observable<Pasien> {
    return this.http.get<Pasien>(`${this.base}/${id}`);
  }

  /**
   * Update biodata pasien (PATCH /pasien/:id). Payload menyertakan version untuk optimistic locking.
   */
  update(id: number, payload: UpdatePasienRequest): Observable<Pasien> {
    return this.http.patch<Pasien>(`${this.base}/${id}`, payload);
  }
}

