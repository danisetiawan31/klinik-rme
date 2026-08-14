import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { formatJakartaDate, getJakartaISODate } from '../../core/utils/date.utils';
import { ToastComponent } from '../../shared/components/toast/toast.component';
import { LaporanService } from './laporan.service';
import { LaporanHarian } from './laporan.types';

@Component({
  selector: 'app-laporan-harian',
  standalone: true,
  imports: [ToastComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './laporan-harian.component.html',
})
export class LaporanHarianComponent implements OnInit {
  private laporanService = inject(LaporanService);

  readonly tanggalFilter = signal<string>(getJakartaISODate());
  readonly laporan = signal<LaporanHarian | null>(null);
  readonly isLoading = signal<boolean>(false);
  readonly errorMessage = signal<string | null>(null);

  ngOnInit(): void {
    this.fetchLaporan(this.tanggalFilter());
  }

  onTanggalChange(newTanggal: string): void {
    if (!newTanggal) return;
    this.tanggalFilter.set(newTanggal);
    this.fetchLaporan(newTanggal);
  }

  fetchLaporan(tanggal: string): void {
    this.isLoading.set(true);
    this.errorMessage.set(null);

    this.laporanService.getLaporanHarian(tanggal).subscribe({
      next: (data) => {
        this.laporan.set(data);
        this.isLoading.set(false);
      },
      error: (err: any) => {
        this.isLoading.set(false);
        const msg =
          err?.error?.message ??
          err?.error?.error?.message ??
          'Gagal memuat laporan harian. Silakan coba lagi.';
        this.errorMessage.set(msg);
      },
    });
  }

  formatDate(isoDateStr: string): string {
    if (!isoDateStr) return '-';
    try {
      return formatJakartaDate(isoDateStr);
    } catch {
      return isoDateStr;
    }
  }
}
