import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-antrian-dashboard',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="min-h-screen bg-[var(--color-background)] p-8">
      <div class="max-w-4xl mx-auto bg-[var(--color-card)] rounded-[var(--radius-md)] shadow-[var(--shadow-2)] border border-[var(--color-border)] p-6 flex items-center justify-between">
        <div>
          <h1 class="text-xl font-bold text-[var(--color-foreground)] mb-1">
            Selamat datang, {{ authService.currentUser()?.nama || 'Pengguna' }}!
          </h1>
          <p class="text-sm text-[var(--color-muted-foreground)]">
            Peranan aktif: <span class="font-semibold text-[var(--color-primary)]">{{ authService.currentUser()?.roles?.join(', ') }}</span>
          </p>
          <span class="inline-block mt-3 px-3 py-1 bg-[var(--color-muted)] text-[var(--color-muted-foreground)] text-xs rounded-[var(--radius-full)] font-medium">
            Placeholder Dashboard Antrian (Backlog Item 15)
          </span>
        </div>
        <button
          type="button"
          (click)="authService.logout().subscribe()"
          class="px-4 py-2 bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] font-medium text-sm rounded-[var(--radius-sm)] transition-all"
        >
          Keluar
        </button>
      </div>
    </div>
  `,
})
export class AntrianDashboardComponent {
  readonly authService = inject(AuthService);
}
