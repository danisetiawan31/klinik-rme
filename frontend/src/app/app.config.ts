import { registerLocaleData } from '@angular/common';
import { DATE_PIPE_DEFAULT_OPTIONS } from '@angular/common';
import localeId from '@angular/common/locales/id';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import {
  ApplicationConfig,
  ErrorHandler,
  LOCALE_ID,
  provideBrowserGlobalErrorListeners,
} from '@angular/core';
import { provideRouter } from '@angular/router';
import * as Sentry from '@sentry/angular';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth/auth.interceptor';
import { environment } from '../environments/environment';

registerLocaleData(localeId);

if (environment.sentryDsn) {
  Sentry.init({
    dsn: environment.sentryDsn,
    environment: environment.production ? 'production' : 'development',
    integrations: [Sentry.browserTracingIntegration()],
    beforeSend(event) {
      // Sanitasi PII & data rekam medis sebelum dikirim ke Sentry
      if (event.request && event.request.data) {
        event.request.data = '[FILTERED]';
      }
      return event;
    },
  });
}

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
    ...(environment.sentryDsn
      ? [{ provide: ErrorHandler, useValue: Sentry.createErrorHandler() }]
      : []),
  ],
};
