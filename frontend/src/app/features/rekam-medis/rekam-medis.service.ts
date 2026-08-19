import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import {
  CreateAddendumDto,
  CreateRekamMedisDto,
  RekamMedis,
  RiwayatRekamMedisItem,
} from './rekam-medis.types';

@Injectable({ providedIn: 'root' })
export class RekamMedisService {
  private http = inject(HttpClient);
  private api = environment.apiUrl;

  /**
   * Mengambil rekam medis terkini (leaf) untuk kunjungan tertentu.
   * Endpoint: GET /kunjungan/:id/rekam-medis [dokter]
   */
  getRekamMedisByKunjungan(kunjunganId: number): Observable<RekamMedis> {
    return this.http.get<RekamMedis>(`${this.api}/kunjungan/${kunjunganId}/rekam-medis`);
  }

  /**
   * Membuat rekam medis awal untuk kunjungan aktif dokter.
   * Mengubah status kunjungan menjadi 'selesai' dan memicu broadcast WebSocket.
   * Endpoint: POST /kunjungan/:id/rekam-medis [dokter]
   */
  createRekamMedis(kunjunganId: number, payload: CreateRekamMedisDto): Observable<RekamMedis> {
    return this.http.post<RekamMedis>(
      `${this.api}/kunjungan/${kunjunganId}/rekam-medis`,
      payload
    );
  }

  /**
   * Membuat addendum koreksi rekam medis terhadap leaf record sebelumnya.
   * Endpoint: POST /rekam-medis/:id/addendum [dokter]
   */
  createAddendum(rekamMedisId: number, payload: CreateAddendumDto): Observable<RekamMedis> {
    return this.http.post<RekamMedis>(
      `${this.api}/rekam-medis/${rekamMedisId}/addendum`,
      payload
    );
  }

  /**
   * Mengambil seluruh riwayat rekam medis klinis dari pasien tertentu.
   * Endpoint: GET /pasien/:id/riwayat [dokter]
   */
  getRiwayatByPasien(pasienId: number): Observable<RiwayatRekamMedisItem[]> {
    return this.http.get<RiwayatRekamMedisItem[]>(`${this.api}/pasien/${pasienId}/riwayat`);
  }
}
