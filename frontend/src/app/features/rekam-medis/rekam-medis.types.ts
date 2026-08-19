export type JenisTindakan = 'tindakan' | 'resep';

export interface DiagnosisItem {
  id: number;
  kodeIcd: string | null;
  deskripsi: string;
}

export interface TindakanItem {
  id: number;
  jenis: JenisTindakan;
  deskripsi: string;
}

export interface RekamMedis {
  id: number;
  keluhan: string;
  hasilPemeriksaan: string;
  diagnosis: DiagnosisItem[];
  tindakan: TindakanItem[];
  isAddendum?: boolean;
  addendumOf?: number | null;
  createdAt: string;
}

export interface RiwayatRekamMedisItem {
  kunjunganId: number;
  tanggal: string;
  rekamMedis: RekamMedis;
}

export interface CreateDiagnosisDto {
  kodeIcd?: string | null;
  deskripsi: string;
}

export interface CreateTindakanDto {
  jenis: JenisTindakan;
  deskripsi: string;
}

export interface CreateRekamMedisDto {
  keluhan: string;
  hasilPemeriksaan: string;
  diagnosis: CreateDiagnosisDto[];
  tindakan: CreateTindakanDto[];
}

export interface CreateAddendumDto {
  alasanAddendum: string;
  keluhan?: string;
  hasilPemeriksaan?: string;
  diagnosis?: CreateDiagnosisDto[];
  tindakan?: CreateTindakanDto[];
}
