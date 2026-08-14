import { Routes } from '@angular/router';
import { roleGuard } from '../../core/guards/role.guard';

export const pasienRoutes: Routes = [
  {
    path: 'baru',
    canActivate: [roleGuard('petugas', 'admin')],
    loadComponent: () =>
      import('./components/pasien-form/pasien-form.component').then(
        (m) => m.PasienFormComponent
      ),
  },
];
