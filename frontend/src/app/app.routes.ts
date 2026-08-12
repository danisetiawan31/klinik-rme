import { Routes } from '@angular/router';
import { roleGuard } from './core/guards/role.guard';
import { staffAuthResolver } from './core/guards/staff-auth.resolver';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () =>
      import('./features/auth/login/login.component').then((m) => m.LoginComponent),
  },
  {
    path: '',
    resolve: { user: staffAuthResolver },
    children: [
      {
        path: 'antrian',
        canActivate: [roleGuard('petugas', 'dokter')],
        loadComponent: () =>
          import('./features/antrian/antrian-dashboard.component').then(
            (m) => m.AntrianDashboardComponent
          ),
      },
      {
        path: 'admin',
        canActivate: [roleGuard('admin')],
        loadComponent: () =>
          import('./features/admin/admin-dashboard.component').then(
            (m) => m.AdminDashboardComponent
          ),
      },
      {
        path: '',
        pathMatch: 'full',
        redirectTo: 'login',
      },
    ],
  },
  {
    path: '**',
    redirectTo: 'login',
  },
];
