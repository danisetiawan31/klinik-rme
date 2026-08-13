import { registerLocaleData } from '@angular/common';
import { DATE_PIPE_DEFAULT_OPTIONS } from '@angular/common';
import localeId from '@angular/common/locales/id';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import {
  ApplicationConfig,
  LOCALE_ID,
  provideBrowserGlobalErrorListeners,
} from '@angular/core';
import { provideRouter } from '@angular/router';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth/auth.interceptor';
import { environment } from '../environments/environment';

registerLocaleData(localeId);

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes),
    provideHttpClient(withInterceptors([authInterceptor])),
    { provide: LOCALE_ID, useValue: 'id-ID' },
    {
      provide: DATE_PIPE_DEFAULT_OPTIONS,
      useValue: { timezone: environment.timezone },
    },
  ],
};
