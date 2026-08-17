import { Directive, input } from '@angular/core';
import { classes } from '@spartan-ng/helm/utils';
import { type VariantProps, cva } from 'class-variance-authority';

@Directive({
  selector: '[hlmAlertAction]',
  host: {
    'data-slot': 'alert-action',
  },
})
export class HlmAlertAction {
  constructor() {
    classes(() => 'absolute end-3 top-2.5');
  }
}

@Directive({
  selector: '[hlmAlertDescription]',
  host: {
    'data-slot': 'alert-description',
  },
})
export class HlmAlertDescription {
  constructor() {
    classes(
      () =>
        'text-muted-foreground text-sm leading-relaxed text-balance md:text-pretty [&_p:not(:last-child)]:mb-2 [&_a]:hover:text-foreground [&_a]:underline [&_a]:underline-offset-3',
    );
  }
}

@Directive({
  selector: '[hlmAlertTitle]',
  host: {
    'data-slot': 'alert-title',
  },
})
export class HlmAlertTitle {
  constructor() {
    classes(
      () =>
        'font-semibold text-foreground text-sm leading-none tracking-tight group-has-[>svg]/alert:col-start-2 [&_a]:hover:text-foreground [&_a]:underline [&_a]:underline-offset-3',
    );
  }
}

const alertVariants = cva(
  'grid gap-1 rounded-md border p-4 text-start text-sm has-data-[slot=alert-action]:relative has-data-[slot=alert-action]:pe-24 has-[>svg]:grid-cols-[auto_1fr] has-[>svg]:gap-x-3 *:[svg]:row-span-2 *:[svg]:translate-y-0.5 *:[svg]:text-current group/alert relative w-full',
  {
    variants: {
      variant: {
        default: 'bg-card text-card-foreground border-border',
        destructive:
          'border-destructive/50 bg-destructive/10 text-destructive dark:border-destructive *:data-[slot=alert-description]:text-destructive/90 *:[svg]:text-destructive',
        warning:
          'border-warning/50 bg-warning/10 text-warning-foreground *:data-[slot=alert-description]:text-warning-foreground/90 *:[svg]:text-warning',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
);

export type AlertVariants = VariantProps<typeof alertVariants>;

@Directive({
  selector: 'hlm-alert,[hlmAlert]',
  host: {
    'data-slot': 'alert',
    role: 'alert',
  },
})
export class HlmAlert {
  public readonly variant = input<AlertVariants['variant']>('default');

  constructor() {
    classes(() => alertVariants({ variant: this.variant() }));
  }
}

export const HlmAlertImports = [
  HlmAlert,
  HlmAlertAction,
  HlmAlertDescription,
  HlmAlertTitle,
] as const;
