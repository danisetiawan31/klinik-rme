import { Routes } from '@angular/router';
import { roleGuard } from '../../core/guards/role.guard';

export const pasienRoutes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    canActivate: [roleGuard('petugas', 'dokter', 'admin')],
    loadComponent: () =>
      import('./components/pasien-list/pasien-list.component').then(
        (m) => m.PasienListComponent
      ),
  },
  {
    path: 'baru',
    canActivate: [roleGuard('petugas', 'admin')],
    loadComponent: () =>
      import('./components/pasien-form/pasien-form.component').then(
        (m) => m.PasienFormComponent
      ),
  },
  {
    path: ':id',
    canActivate: [roleGuard('petugas', 'dokter', 'admin')],
    loadComponent: () =>
      import('./components/pasien-detail/pasien-detail.component').then(
        (m) => m.PasienDetailComponent
      ),
  },
];
