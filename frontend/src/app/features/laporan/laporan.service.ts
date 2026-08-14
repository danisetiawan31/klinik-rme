import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { LaporanHarian } from './laporan.types';

@Injectable({ providedIn: 'root' })
export class LaporanService {
  private http = inject(HttpClient);

  /**
   * Mengambil data rekapitulasi laporan harian klinik (GET /api/v1/laporan/harian?tanggal=)
   * Jika parameter tanggal tidak diisi, backend default ke tanggal hari ini berbasis Asia/Jakarta.
   */
  getLaporanHarian(tanggal?: string): Observable<LaporanHarian> {
    let params = new HttpParams();
    if (tanggal) {
      params = params.set('tanggal', tanggal);
    }
    return this.http.get<LaporanHarian>(`${environment.apiUrl}/laporan/harian`, {
      params,
    });
  }
}
