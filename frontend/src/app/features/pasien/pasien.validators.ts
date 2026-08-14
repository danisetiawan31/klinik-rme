import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';

/** Validator: NIK boleh kosong (nullable), tapi kalau diisi wajib persis 16 digit angka */
export function nikFormatValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    const val: string = control.value ?? '';
    if (!val) return null; // kosong = nullable, valid
    return /^\d{16}$/.test(val) ? null : { nikFormat: true };
  };
}
