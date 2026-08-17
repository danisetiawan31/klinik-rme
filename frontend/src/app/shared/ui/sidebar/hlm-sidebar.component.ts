import {
  ChangeDetectionStrategy,
  Component,
  DOCUMENT,
  DestroyRef,
  Directive,
  Injectable,
  InjectionToken,
  PLATFORM_ID,
  REQUEST,
  afterNextRender,
  booleanAttribute,
  computed,
  effect,
  inject,
  input,
  signal,
  type Signal,
  type ValueProvider,
} from '@angular/core';
import { NgTemplateOutlet, isPlatformServer } from '@angular/common';
import { type BooleanInput } from '@angular/cdk/coercion';
import { type ClassValue } from 'clsx';
import { cva } from 'class-variance-authority';
import { NgIcon, provideIcons } from '@ng-icons/core';
import { lucidePanelLeft } from '@ng-icons/lucide';
import { HlmButton } from '../button/src/lib/hlm-button';
import { HlmIconDirective } from '../icon/src/lib/hlm-icon.directive';
import { HlmSheetImports } from '@spartan-ng/helm/sheet';
import { classes, hlm } from '@spartan-ng/helm/utils';

export type SidebarVariant = 'sidebar' | 'floating' | 'inset';

export interface HlmSidebarConfig {
  defaultOpen: boolean;
  sidebarWidth: string;
  sidebarWidthMobile: string;
  sidebarWidthIcon: string;
  sidebarCookieName: string;
  sidebarCookieMaxAge: number;
  sidebarKeyboardShortcut: string;
  mobileBreakpoint: string;
  closeMobileSidebarOnMenuButtonClick: boolean;
}

const defaultConfig: HlmSidebarConfig = {
  defaultOpen: true,
  sidebarWidth: '16rem',
  sidebarWidthMobile: '18rem',
  sidebarWidthIcon: '3rem',
  sidebarCookieName: 'sidebar_state',
  sidebarCookieMaxAge: 60 * 60 * 24 * 7,
  sidebarKeyboardShortcut: 'b',
  mobileBreakpoint: '768px',
  closeMobileSidebarOnMenuButtonClick: false,
};

const HlmSidebarConfigToken = new InjectionToken<HlmSidebarConfig>('HlmSidebarConfig');

export function provideHlmSidebarConfig(config: Partial<HlmSidebarConfig>): ValueProvider {
  return { provide: HlmSidebarConfigToken, useValue: { ...defaultConfig, ...config } };
}

export function injectHlmSidebarConfig(): HlmSidebarConfig {
  return inject(HlmSidebarConfigToken, { optional: true }) ?? defaultConfig;
}

@Injectable({ providedIn: 'root' })
export class HlmSidebarService {
  private readonly _platformId = inject(PLATFORM_ID);
  private readonly _request = inject(REQUEST, { optional: true });
  private readonly _config = injectHlmSidebarConfig();
  private readonly _document = inject(DOCUMENT);
  private readonly _window = this._document.defaultView;
  private readonly _open = signal<boolean>(this._config.defaultOpen);
  private readonly _openMobile = signal<boolean>(false);
  private readonly _isMobile = signal<boolean>(false);
  private readonly _variant = signal<SidebarVariant>('inset');
  private _mediaQuery: MediaQueryList | null = null;

  public readonly open: Signal<boolean> = this._open.asReadonly();
  public readonly openMobile: Signal<boolean> = this._openMobile.asReadonly();
  public readonly isMobile: Signal<boolean> = this._isMobile.asReadonly();
  public readonly variant: Signal<SidebarVariant> = this._variant.asReadonly();

  public readonly state = computed<'expanded' | 'collapsed'>(() =>
    this._open() ? 'expanded' : 'collapsed'
  );

  constructor() {
    const destroyRef = inject(DestroyRef);
    this.restoreStateFromCookie();

    afterNextRender(() => {
      if (!this._window || typeof this._window.matchMedia !== 'function') return;

      this._mediaQuery = this._window.matchMedia(`(max-width: ${this._config.mobileBreakpoint})`);
      this._isMobile.set(this._mediaQuery.matches);

      const mediaQueryHandler = (e: MediaQueryListEvent) => {
        this._isMobile.set(e.matches);
        if (!e.matches) this._openMobile.set(false);
      };
      this._mediaQuery.addEventListener('change', mediaQueryHandler);

      destroyRef.onDestroy(() => {
        if (this._mediaQuery) this._mediaQuery.removeEventListener('change', mediaQueryHandler);
      });
    });
  }

  public setOpen(open: boolean): void {
    this._open.set(open);
    this._document.cookie = `${this._config.sidebarCookieName}=${open}; path=/; max-age=${this._config.sidebarCookieMaxAge}`;
  }

  public setOpenMobile(open: boolean): void {
    this._openMobile.set(open);
  }

  public setVariant(variant: SidebarVariant): void {
    this._variant.set(variant);
  }

  public toggleSidebar(): void {
    if (this._isMobile()) {
      this._openMobile.update((value) => !value);
    } else {
      this.setOpen(!this._open());
    }
  }

  private restoreStateFromCookie(): void {
    const cookieString = isPlatformServer(this._platformId)
      ? this._request?.headers.get('cookie')
      : this._document.cookie;

    if (!cookieString) return;

    const prefix = `${this._config.sidebarCookieName}=`;
    const cookieValue = cookieString
      .split(';')
      .map((c) => c.trim())
      .find((c) => c.startsWith(prefix))
      ?.slice(prefix.length);

    if (cookieValue !== undefined) {
      this._open.set(cookieValue === 'true');
    }
  }
}

@Directive({
  selector: '[hlmSidebarWrapper],hlm-sidebar-wrapper',
  host: {
    'data-slot': 'sidebar-wrapper',
    '[style.--sidebar-width]': 'sidebarWidth()',
    '[style.--sidebar-width-icon]': 'sidebarWidthIcon()',
  },
})
export class HlmSidebarWrapper {
  private readonly _config = injectHlmSidebarConfig();

  public readonly sidebarWidth = input<string>(this._config.sidebarWidth);
  public readonly sidebarWidthIcon = input<string>(this._config.sidebarWidthIcon);

  constructor() {
    classes(() => 'group/sidebar-wrapper has-data-[variant=inset]:bg-card flex min-h-svh w-full');
  }
}

@Component({
  selector: 'hlm-sidebar',
  imports: [NgTemplateOutlet, HlmSheetImports],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    '[attr.data-slot]': '_dataSlot()',
    '[attr.data-state]': '_dataState()',
    '[attr.data-collapsible]': '_dataCollapsible()',
    '[attr.data-variant]': '_dataVariant()',
    '[attr.data-side]': '_dataSide()',
  },
  template: `
    <ng-template #contentContainer>
      <ng-content />
    </ng-template>

    @if (collapsible() === 'none') {
      <ng-container *ngTemplateOutlet="contentContainer" />
    } @else if (_sidebarService.isMobile()) {
      <hlm-sheet
        [side]="side()"
        [state]="_sidebarService.openMobile() ? 'open' : 'closed'"
        (stateChanged)="_sidebarService.setOpenMobile($event === 'open')"
      >
        <hlm-sheet-content
          *hlmSheetPortal="let ctx"
          data-slot="sidebar"
          data-sidebar="sidebar"
          data-mobile="true"
          class="bg-card text-foreground h-svh w-(--sidebar-width) p-0 [&>button]:hidden"
          [style.--sidebar-width]="sidebarWidthMobile()"
        >
          <div class="flex h-full w-full flex-col">
            <ng-container *ngTemplateOutlet="contentContainer" />
          </div>
        </hlm-sheet-content>
      </hlm-sheet>
    } @else {
      <!-- Sidebar gap on desktop -->
      <div data-slot="sidebar-gap" [class]="_sidebarGapComputedClass()"></div>
      <div data-slot="sidebar-container" [attr.data-side]="_dataSide()" [class]="_sidebarContainerComputedClass()">
        <div data-sidebar="sidebar" data-slot="sidebar-inner" class="bg-card group-data-[variant=inset]:border-r group-data-[variant=inset]:border-border/80 flex size-full flex-col">
          <ng-container *ngTemplateOutlet="contentContainer" />
        </div>
      </div>
    }
  `,
})
export class HlmSidebar {
  protected readonly _sidebarService = inject(HlmSidebarService);
  private readonly _config = injectHlmSidebarConfig();
  public readonly sidebarWidthMobile = input<string>(this._config.sidebarWidthMobile);

  public readonly side = input<'left' | 'right'>('left');
  public readonly variant = input<SidebarVariant>(this._sidebarService.variant());
  public readonly collapsible = input<'offcanvas' | 'icon' | 'none'>('icon');

  protected readonly _sidebarGapComputedClass = computed(() =>
    hlm(
      'transition-[width] duration-200 ease-linear relative w-(--sidebar-width) bg-transparent',
      'group-data-[collapsible=offcanvas]:w-0',
      'group-data-[side=right]:rotate-180',
      this.variant() === 'floating' || this.variant() === 'inset'
        ? 'group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4)))]'
        : 'group-data-[collapsible=icon]:w-(--sidebar-width-icon)'
    )
  );

  public readonly sidebarContainerClass = input<ClassValue>('');
  protected readonly _sidebarContainerComputedClass = computed(() =>
    hlm(
      'fixed inset-y-0 z-20 hidden h-svh w-(--sidebar-width) transition-[left,right,width] duration-200 ease-linear data-[side=left]:left-0 data-[side=left]:group-data-[collapsible=offcanvas]:left-[calc(var(--sidebar-width)*-1)] data-[side=right]:right-0 data-[side=right]:group-data-[collapsible=offcanvas]:right-[calc(var(--sidebar-width)*-1)] md:flex',
      this.variant() === 'floating' || this.variant() === 'inset'
        ? 'p-2 group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4))+2px)]'
        : 'group-data-[collapsible=icon]:w-(--sidebar-width-icon) group-data-[side=left]:border-r group-data-[side=right]:border-l',
      this.sidebarContainerClass()
    )
  );

  protected readonly _dataSlot = computed(() => (!this._sidebarService.isMobile() ? 'sidebar' : undefined));

  private readonly _collapsibleAndNonMobile = computed(
    () => this.collapsible() !== 'none' && !this._sidebarService.isMobile()
  );

  protected readonly _dataState = computed(() =>
    this._collapsibleAndNonMobile() ? this._sidebarService.state() : undefined
  );

  protected readonly _dataCollapsible = computed(() => {
    if (this._collapsibleAndNonMobile()) {
      return this._sidebarService.state() === 'collapsed' ? this.collapsible() : '';
    }
    return undefined;
  });

  protected readonly _dataVariant = computed(() =>
    this._collapsibleAndNonMobile() ? this.variant() : undefined
  );

  protected readonly _dataSide = computed(() =>
    this._collapsibleAndNonMobile() ? this.side() : undefined
  );

  constructor() {
    effect(() => {
      this._sidebarService.setVariant(this.variant());
    });

    classes(() => {
      if (this.collapsible() === 'none') {
        return hlm('bg-card text-foreground flex h-svh w-(--sidebar-width) flex-col');
      } else if (this._sidebarService.isMobile()) {
        return '';
      } else {
        return hlm('group peer text-foreground hidden md:block');
      }
    });
  }
}

@Directive({
  selector: '[hlmSidebarHeader],hlm-sidebar-header',
  host: {
    'data-slot': 'sidebar-header',
    'data-sidebar': 'header',
  },
})
export class HlmSidebarHeader {
  constructor() {
    classes(() => 'gap-2 p-2.5 flex flex-col border-b border-border/70 group-data-[collapsible=icon]:p-2 group-data-[collapsible=icon]:items-center');
  }
}

@Directive({
  selector: '[hlmSidebarContent],hlm-sidebar-content',
  host: {
    'data-slot': 'sidebar-content',
    'data-sidebar': 'content',
  },
})
export class HlmSidebarContent {
  constructor() {
    classes(() => 'gap-0 flex min-h-0 flex-1 flex-col overflow-y-auto group-data-[collapsible=icon]:overflow-hidden');
  }
}

@Directive({
  selector: '[hlmSidebarFooter],hlm-sidebar-footer',
  host: {
    'data-slot': 'sidebar-footer',
    'data-sidebar': 'footer',
  },
})
export class HlmSidebarFooter {
  constructor() {
    classes(() => 'gap-2 p-2 flex flex-col border-t border-border/70 mt-auto bg-card group-data-[collapsible=icon]:p-2 group-data-[collapsible=icon]:items-center');
  }
}

@Directive({
  selector: '[hlmSidebarGroup],hlm-sidebar-group',
  host: {
    'data-slot': 'sidebar-group',
    'data-sidebar': 'group',
  },
})
export class HlmSidebarGroup {
  constructor() {
    classes(() => 'p-2 relative flex w-full min-w-0 flex-col group-data-[collapsible=icon]:p-1 group-data-[collapsible=icon]:items-center');
  }
}

@Directive({
  selector: 'div[hlmSidebarGroupLabel], button[hlmSidebarGroupLabel]',
  host: {
    'data-slot': 'sidebar-group-label',
    'data-sidebar': 'group-label',
  },
})
export class HlmSidebarGroupLabel {
  constructor() {
    classes(
      () =>
        'text-muted-foreground/80 h-7 rounded-md px-2 text-[11px] font-bold uppercase tracking-wider transition-[margin,opacity] duration-200 ease-linear group-data-[collapsible=icon]:hidden flex shrink-0 items-center select-none'
    );
  }
}

@Directive({
  selector: 'div[hlmSidebarGroupContent]',
  host: {
    'data-slot': 'sidebar-group-content',
    'data-sidebar': 'group-content',
  },
})
export class HlmSidebarGroupContent {
  constructor() {
    classes(() => 'text-sm w-full');
  }
}

@Directive({
  selector: 'ul[hlmSidebarMenu]',
  host: {
    'data-slot': 'sidebar-menu',
    'data-sidebar': 'menu',
  },
})
export class HlmSidebarMenu {
  constructor() {
    classes(() => 'gap-1 flex w-full min-w-0 flex-col list-none p-0 m-0 group-data-[collapsible=icon]:items-center');
  }
}

@Directive({
  selector: 'li[hlmSidebarMenuItem]',
  host: {
    'data-slot': 'sidebar-menu-item',
    'data-sidebar': 'menu-item',
  },
})
export class HlmSidebarMenuItem {
  constructor() {
    classes(() => 'group/menu-item relative list-none w-full group-data-[collapsible=icon]:flex group-data-[collapsible=icon]:justify-center');
  }
}

const sidebarMenuButtonVariants = cva(
  'hover:bg-muted hover:text-foreground active:bg-muted/90 data-[active=true]:bg-primary/10 data-[active=true]:text-primary data-[active=true]:font-semibold gap-2.5 rounded-lg px-2.5 py-2 text-start text-sm transition-all duration-150 flex w-full items-center overflow-hidden outline-hidden cursor-pointer select-none disabled:pointer-events-none disabled:opacity-50 group-data-[collapsible=icon]:size-8! group-data-[collapsible=icon]:p-0! group-data-[collapsible=icon]:justify-center [&>span:last-child]:truncate',
  {
    variants: {
      variant: {
        default: '',
        outline: 'bg-background hover:bg-muted border border-border',
      },
      size: {
        default: 'h-9 text-sm',
        sm: 'h-8 text-xs',
        lg: 'h-12 text-sm',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
);

@Directive({
  selector: 'button[hlmSidebarMenuButton], a[hlmSidebarMenuButton]',
  host: {
    'data-slot': 'sidebar-menu-button',
    'data-sidebar': 'menu-button',
    '[attr.data-size]': 'size()',
    '[attr.data-active]': 'isActive()',
  },
})
export class HlmSidebarMenuButton {
  public readonly variant = input<'default' | 'outline'>('default');
  public readonly size = input<'default' | 'sm' | 'lg'>('default');
  public readonly isActive = input<boolean, BooleanInput>(false, { transform: booleanAttribute });

  constructor() {
    classes(() => sidebarMenuButtonVariants({ variant: this.variant(), size: this.size() }));
  }
}

@Directive({
  selector: 'main[hlmSidebarInset]',
  host: { 'data-slot': 'sidebar-inset' },
})
export class HlmSidebarInset {
  constructor() {
    classes(
      () =>
        'bg-background relative flex w-full flex-1 flex-col overflow-y-auto md:peer-data-[variant=inset]:m-2 md:peer-data-[variant=inset]:ms-0 md:peer-data-[variant=inset]:rounded-xl md:peer-data-[variant=inset]:border md:peer-data-[variant=inset]:border-border md:peer-data-[variant=inset]:shadow-xs'
    );
  }
}

@Component({
  selector: 'button[hlmSidebarTrigger]',
  standalone: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [NgIcon, HlmIconDirective],
  providers: [provideIcons({ lucidePanelLeft })],
  hostDirectives: [{ directive: HlmButton, inputs: ['variant', 'size'] }],
  host: {
    'data-slot': 'sidebar-trigger',
    'data-sidebar': 'trigger',
    '(click)': '_onClick()',
  },
  template: `
    <ng-icon hlm size="base" name="lucidePanelLeft" aria-hidden="true" />
    <span class="sr-only">Toggle Sidebar</span>
  `,
})
export class HlmSidebarTrigger {
  private readonly _sidebarService = inject(HlmSidebarService);

  protected _onClick(): void {
    this._sidebarService.toggleSidebar();
  }
}

@Directive({
  selector: 'button[hlmSidebarRail]',
  host: {
    'data-sidebar': 'rail',
    'data-slot': 'sidebar-rail',
    tabindex: '-1',
    '(click)': 'onClick()',
  },
})
export class HlmSidebarRail {
  private readonly _sidebarService = inject(HlmSidebarService);

  constructor() {
    classes(() => [
      'hover:after:bg-border absolute inset-y-0 z-20 hidden w-4 transition-all ease-linear group-data-[side=left]:-right-4 group-data-[side=right]:left-0 after:absolute after:inset-y-0 after:start-1/2 after:w-[2px] sm:flex ltr:-translate-x-1/2 rtl:-translate-x-1/2 in-data-[side=left]:cursor-w-resize in-data-[side=right]:cursor-e-resize',
    ]);
  }

  protected onClick(): void {
    this._sidebarService.toggleSidebar();
  }
}

export const HlmSidebarImports = [
  HlmSidebar,
  HlmSidebarContent,
  HlmSidebarFooter,
  HlmSidebarGroup,
  HlmSidebarGroupContent,
  HlmSidebarGroupLabel,
  HlmSidebarHeader,
  HlmSidebarInset,
  HlmSidebarMenu,
  HlmSidebarMenuItem,
  HlmSidebarMenuButton,
  HlmSidebarRail,
  HlmSidebarTrigger,
  HlmSidebarWrapper,
] as const;
