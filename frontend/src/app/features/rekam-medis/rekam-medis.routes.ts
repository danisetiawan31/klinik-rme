import { Routes } from '@angular/router';

export const rekamMedisRoutes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    redirectTo: '/antrian',
  },
  {
    path: 'pemeriksaan/:kunjunganId',
    loadComponent: () =>
      import('./components/rekam-medis-form/rekam-medis-form.component').then(
        (m) => m.RekamMedisFormComponent
      ),
  },
  {
    path: 'kunjungan/:kunjunganId',
    loadComponent: () =>
      import('./components/rekam-medis-detail/rekam-medis-detail.component').then(
        (m) => m.RekamMedisDetailComponent
      ),
  },
];
