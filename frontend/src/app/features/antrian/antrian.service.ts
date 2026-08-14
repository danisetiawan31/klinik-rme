import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { KunjunganListItem, PanggilBerikutnyaResponse } from './antrian.types';

@Injectable({ providedIn: 'root' })
export class AntrianService {
  private http = inject(HttpClient);

  /**
   * Fetch daftar antrian aktif klinik hari ini (GET /api/v1/klinik/:id/antrian)
   */
  getAntrian(klinikId: number = environment.defaultKlinikId): Observable<KunjunganListItem[]> {
    return this.http.get<KunjunganListItem[]>(`${environment.apiUrl}/klinik/${klinikId}/antrian`);
  }

  /**
   * Panggil pasien antrian berikutnya untuk dokter (POST /api/v1/klinik/:id/panggil-berikutnya)
   * Mengembalikan null jika respons HTTP adalah 204 No Content (antrian kosong).
   */
  panggilBerikutnya(klinikId: number = environment.defaultKlinikId): Observable<PanggilBerikutnyaResponse | null> {
    return this.http
      .post<PanggilBerikutnyaResponse>(
        `${environment.apiUrl}/klinik/${klinikId}/panggil-berikutnya`,
        {},
        { observe: 'response' }
      )
      .pipe(
        map((res) => {
          if (res.status === 204) {
            return null;
          }
          return res.body;
        })
      );
  }

  /**
   * Dokter melewati pasien yang tidak muncul saat dipanggil (POST /api/v1/kunjungan/:id/lewati)
   * Pasien kembali berstatus 'menunggu' dengan penambahan skipCount.
   */
  lewati(kunjunganId: number): Observable<{ id: number; status: string; skipCount: number }> {
    return this.http.post<{ id: number; status: string; skipCount: number }>(
      `${environment.apiUrl}/kunjungan/${kunjunganId}/lewati`,
      {}
    );
  }

  /**
   * Dokter / Admin menandai pasien tidak hadir secara final (POST /api/v1/kunjungan/:id/tidak-hadir)
   */
  tidakHadir(kunjunganId: number): Observable<{ id: number; status: string }> {
    return this.http.post<{ id: number; status: string }>(
      `${environment.apiUrl}/kunjungan/${kunjunganId}/tidak-hadir`,
      {}
    );
  }
}
