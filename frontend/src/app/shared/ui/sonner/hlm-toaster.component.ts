import {
  ChangeDetectionStrategy,
  Component,
  booleanAttribute,
  computed,
  input,
  numberAttribute,
} from '@angular/core';
import { type BooleanInput, type NumberInput } from '@angular/cdk/coercion';
import { type ClassValue } from 'clsx';
import { NgIcon, provideIcons } from '@ng-icons/core';
import {
  lucideCircleCheck,
  lucideInfo,
  lucideLoader2,
  lucideOctagonX,
  lucideTriangleAlert,
} from '@ng-icons/lucide';
import { BrnSonnerImports, type ToasterProps } from '@spartan-ng/brain/sonner';
import { hlm } from '@spartan-ng/helm/utils';

@Component({
  selector: 'hlm-toaster',
  standalone: true,
  imports: [BrnSonnerImports, NgIcon],
  providers: [
    provideIcons({
      lucideCircleCheck,
      lucideInfo,
      lucideTriangleAlert,
      lucideOctagonX,
      lucideLoader2,
    }),
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <brn-sonner-toaster
      [class]="_computedClass()"
      [invert]="invert()"
      [theme]="theme()"
      [position]="position()"
      [hotKey]="hotKey()"
      [richColors]="richColors()"
      [expand]="expand()"
      [duration]="duration()"
      [visibleToasts]="visibleToasts()"
      [closeButton]="closeButton()"
      [toastOptions]="_computedToastOptions()"
      [offset]="offset()"
      [style]="userStyle()"
    >
      <ng-template #loadingIcon>
        <ng-icon name="lucideLoader2" class="overflow-visible! text-base [&>svg]:motion-safe:animate-spin text-primary" />
      </ng-template>
      <ng-template #successIcon>
        <ng-icon name="lucideCircleCheck" class="overflow-visible! text-base text-accent" />
      </ng-template>
      <ng-template #errorIcon>
        <ng-icon name="lucideOctagonX" class="overflow-visible! text-base text-destructive" />
      </ng-template>
      <ng-template #infoIcon>
        <ng-icon name="lucideInfo" class="overflow-visible! text-base text-primary" />
      </ng-template>
      <ng-template #warningIcon>
        <ng-icon name="lucideTriangleAlert" class="overflow-visible! text-base text-warning" />
      </ng-template>
    </brn-sonner-toaster>
  `,
})
export class HlmToaster {
  public readonly invert = input<ToasterProps['invert'], BooleanInput>(false, {
    transform: booleanAttribute,
  });
  public readonly theme = input<ToasterProps['theme']>('light');
  public readonly position = input<ToasterProps['position']>('top-right');
  public readonly hotKey = input<ToasterProps['hotkey']>(['altKey', 'KeyT']);
  public readonly richColors = input<ToasterProps['richColors'], BooleanInput>(false, {
    transform: booleanAttribute,
  });
  public readonly expand = input<ToasterProps['expand'], BooleanInput>(false, {
    transform: booleanAttribute,
  });
  public readonly duration = input<ToasterProps['duration'], NumberInput>(4000, {
    transform: numberAttribute,
  });
  public readonly visibleToasts = input<ToasterProps['visibleToasts'], NumberInput>(3, {
    transform: numberAttribute,
  });
  public readonly closeButton = input<ToasterProps['closeButton'], BooleanInput>(true, {
    transform: booleanAttribute,
  });
  public readonly toastOptions = input<ToasterProps['toastOptions']>({});

  protected readonly _computedToastOptions = computed(() => {
    const options = this.toastOptions();
    return {
      ...options,
      classes: {
        ...options?.classes,
        toast: hlm(
          'rounded-xl! border border-border/80 bg-card text-foreground shadow-3 px-4 py-3 text-xs font-medium font-sans',
          options?.classes?.toast
        ),
        error: hlm('border-destructive/40 text-destructive', options?.classes?.error),
        success: hlm('border-accent/40 text-foreground', options?.classes?.success),
        info: hlm('border-primary/40 text-foreground', options?.classes?.info),
        warning: hlm('border-warning/40 text-foreground', options?.classes?.warning),
      },
    };
  });

  public readonly offset = input<ToasterProps['offset']>(null);
  public readonly userClass = input<ClassValue>('', { alias: 'class' });
  public readonly userStyle = input<Record<string, string>>(
    {
      '--normal-bg': 'var(--card)',
      '--normal-text': 'var(--foreground)',
      '--normal-border': 'var(--border)',
      '--border-radius': 'var(--radius-md)',
    },
    { alias: 'style' }
  );

  protected readonly _computedClass = computed(() => hlm('toaster group', this.userClass()));
}

export const HlmToasterImports = [HlmToaster] as const;
