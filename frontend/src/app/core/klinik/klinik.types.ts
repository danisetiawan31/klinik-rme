export interface KlinikResponse {
  id: number;
  nama: string;
  alamat?: string;
  telepon?: string;
  jamBuka?: string;
  jamTutup?: string;
  isBuka?: boolean;
}
