import { Directive, input } from '@angular/core';
import { classes } from '@spartan-ng/helm/utils';
import { type VariantProps, cva } from 'class-variance-authority';

@Directive({
  selector: '[hlmEmptyContent],hlm-empty-content',
  host: { 'data-slot': 'empty-content' },
})
export class HlmEmptyContent {
  constructor() {
    classes(
      () =>
        'gap-3 text-sm flex w-full max-w-sm min-w-0 flex-col items-center text-balance',
    );
  }
}

@Directive({
  selector: '[hlmEmptyDescription]',
  host: { 'data-slot': 'empty-description' },
})
export class HlmEmptyDescription {
  constructor() {
    classes(
      () =>
        'text-xs sm:text-sm leading-relaxed text-muted-foreground [&>a:hover]:text-primary [&>a]:underline [&>a]:underline-offset-4',
    );
  }
}

@Directive({
  selector: '[hlmEmptyHeader],hlm-empty-header',
  host: { 'data-slot': 'empty-header' },
})
export class HlmEmptyHeader {
  constructor() {
    classes(() => 'gap-2 flex max-w-sm flex-col items-center');
  }
}

const emptyMediaVariants = cva(
  'mb-2 flex shrink-0 items-center justify-center [&_ng-icon]:pointer-events-none [&_ng-icon]:shrink-0 [&_svg]:pointer-events-none [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        default: 'bg-transparent',
        icon: 'bg-muted text-foreground flex size-10 shrink-0 items-center justify-center rounded-xl border border-border/50 shadow-2xs',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
);

export type EmptyMediaVariants = VariantProps<typeof emptyMediaVariants>;

@Directive({
  selector: '[hlmEmptyMedia],hlm-empty-media',
  host: {
    'data-slot': 'empty-media',
    '[attr.data-variant]': 'variant()',
  },
})
export class HlmEmptyMedia {
  public readonly variant = input<EmptyMediaVariants['variant']>('default');

  constructor() {
    classes(() => emptyMediaVariants({ variant: this.variant() }));
  }
}

@Directive({
  selector: '[hlmEmptyTitle]',
  host: { 'data-slot': 'empty-title' },
})
export class HlmEmptyTitle {
  constructor() {
    classes(() => 'text-sm font-semibold tracking-tight text-foreground');
  }
}

@Directive({
  selector: '[hlmEmpty],hlm-empty',
  host: { 'data-slot': 'empty' },
})
export class HlmEmpty {
  constructor() {
    classes(
      () =>
        'gap-4 rounded-xl border border-dashed border-border/70 p-8 sm:p-12 flex w-full min-w-0 flex-1 flex-col items-center justify-center text-center text-balance',
    );
  }
}

export const HlmEmptyImports = [
  HlmEmpty,
  HlmEmptyContent,
  HlmEmptyDescription,
  HlmEmptyHeader,
  HlmEmptyTitle,
  HlmEmptyMedia,
] as const;
