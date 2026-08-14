export interface Pasien {
  id: number;
  nik: string | null;
  nama: string;
  tanggalLahir: string;
  jenisKelamin: 'L' | 'P';
  alamat: string;
  noTelp: string;
  consent: boolean;
  version: number;
  riwayatKunjunganRingkas: RiwayatKunjunganRingkas[];
}

export interface RiwayatKunjunganRingkas {
  kunjunganId: number;
  tanggal: string;
  status: 'menunggu' | 'dipanggil' | 'selesai' | 'tidak_hadir';
}

export interface PasienSearchItem {
  id: number;
  nik: string | null;
  nama: string;
  tanggalLahir: string;
}

export interface CreatePasienRequest {
  nik?: string | null;
  nama: string;
  tanggalLahir: string;
  jenisKelamin: 'L' | 'P';
  alamat: string;
  noTelp: string;
  consent: boolean;
}

export interface UpdatePasienRequest {
  nik?: string | null;
  nama?: string;
  tanggalLahir?: string;
  jenisKelamin?: 'L' | 'P';
  alamat?: string;
  noTelp?: string;
  version: number;
}
