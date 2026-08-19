export type KunjunganStatus = 'menunggu' | 'dipanggil' | 'selesai' | 'tidak_hadir';

export interface KunjunganListItem {
  id: number;
  nomorAntrian: number;
  status: KunjunganStatus;
  isPriority: boolean;
  priorityReason?: string | null;
  skipCount?: number;
  pasienNama: string;
}

export interface CreateKunjunganRequest {
  pasienId: number;
  isPriority?: boolean;
  priorityReason?: string;
}

export interface CreateKunjunganResponse {
  id: number;
  nomorAntrian: number;
  status: 'menunggu';
  tanggalKunjungan: string;
}

export interface PanggilBerikutnyaResponse {
  id: number;
  nomorAntrian: number;
  pasienNama: string;
  dokterId: number;
  dipanggilAt: string;
}

export interface KunjunganDetail {
  id: number;
  pasienId: number;
  nomorAntrian: number;
  status: KunjunganStatus;
  isPriority: boolean;
  dokterId?: number | null;
  dipanggilAt?: string | null;
}
