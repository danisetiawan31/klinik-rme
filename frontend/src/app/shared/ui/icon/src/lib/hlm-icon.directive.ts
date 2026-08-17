import { Directive, input } from '@angular/core';
import { classes } from '@spartan-ng/helm/utils';
import { type VariantProps, cva } from 'class-variance-authority';

const iconVariants = cva('inline-flex shrink-0 items-center justify-center align-middle', {
  variants: {
    size: {
      xs: 'size-3.5 [&>svg]:size-3.5',
      sm: 'size-4 [&>svg]:size-4',
      base: 'size-5 [&>svg]:size-5',
      lg: 'size-6 [&>svg]:size-6',
      xl: 'size-8 [&>svg]:size-8',
      custom: '',
    },
  },
  defaultVariants: {
    size: 'base',
  },
});

export type IconVariants = VariantProps<typeof iconVariants>;

@Directive({
  selector: 'ng-icon[hlm],ng-icon[hlmIcon]',
  host: {
    '[attr.data-slot]': "'icon'",
  },
})
export class HlmIconDirective {
  public readonly size = input<IconVariants['size']>('base');

  constructor() {
    classes(() => iconVariants({ size: this.size() }));
  }
}
