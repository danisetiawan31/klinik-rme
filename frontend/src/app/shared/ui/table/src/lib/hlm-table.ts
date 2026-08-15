import { Directive } from '@angular/core';
import { classes } from '@spartan-ng/helm/utils';

@Directive({
  selector: 'div[hlmTableContainer]',
  host: { 'data-slot': 'table-container' },
})
export class HlmTableContainer {
  constructor() {
    classes(() => 'relative w-full overflow-x-auto');
  }
}

/**
 * Directive to apply Spartan/Shadcn styling to a <table> element.
 */
@Directive({
  selector: 'table[hlmTable]',
  host: { 'data-slot': 'table' },
})
export class HlmTable {
  constructor() {
    classes(() => 'w-full caption-bottom text-sm border-collapse');
  }
}

/**
 * Directive to apply Spartan/Shadcn styling to a <thead> element
 * within an HlmTable context.
 */
@Directive({
  selector: 'thead[hlmTHead],thead[hlmTableHeader]',
  host: { 'data-slot': 'table-header' },
})
export class HlmTHead {
  constructor() {
    classes(() => 'bg-muted/40 text-muted-foreground border-b border-border/80 [&_tr]:border-b-0');
  }
}

/**
 * Directive to apply Spartan/Shadcn styling to a <tbody> element
 * within an HlmTable context.
 */
@Directive({
  selector: 'tbody[hlmTBody],tbody[hlmTableBody]',
  host: { 'data-slot': 'table-body' },
})
export class HlmTBody {
  constructor() {
    classes(() => '[&_tr:last-child]:border-0');
  }
}

/**
 * Directive to apply Spartan/Shadcn styling to a <tfoot> element
 * within an HlmTable context.
 */
@Directive({
  selector: 'tfoot[hlmTFoot],tfoot[hlmTableFooter]',
  host: { 'data-slot': 'table-footer' },
})
export class HlmTFoot {
  constructor() {
    classes(() => 'bg-muted/40 border-t border-border/80 font-medium [&>tr]:last:border-b-0');
  }
}

/**
 * Directive to apply Spartan/Shadcn styling to a <tr> element
 * within an HlmTable context.
 */
@Directive({
  selector: 'tr[hlmTr],tr[hlmTableRow]',
  host: { 'data-slot': 'table-row' },
})
export class HlmTr {
  constructor() {
    classes(() => 'hover:bg-muted/30 data-[state=selected]:bg-muted border-b border-border/60 transition-colors');
  }
}

/**
 * Directive to apply Spartan/Shadcn styling to a <th> element
 * within an HlmTable context.
 */
@Directive({
  selector: 'th[hlmTh],th[hlmTableHead]',
  host: { 'data-slot': 'table-head' },
})
export class HlmTh {
  constructor() {
    classes(() => 'text-muted-foreground text-xs font-semibold uppercase tracking-wider h-11 px-4 text-start align-middle whitespace-nowrap first:pl-6 last:pr-6 [&:has([role=checkbox])]:pe-0');
  }
}

/**
 * Directive to apply Spartan/Shadcn styling to a <td> element
 * within an HlmTable context.
 */
@Directive({
  selector: 'td[hlmTd],td[hlmTableCell]',
  host: { 'data-slot': 'table-cell' },
})
export class HlmTd {
  constructor() {
    classes(() => 'px-4 py-3.5 align-middle text-sm whitespace-nowrap first:pl-6 last:pr-6 [&:has([role=checkbox])]:pe-0');
  }
}

/**
 * Directive to apply Spartan/Shadcn styling to a <caption> element
 * within an HlmTable context.
 */
@Directive({
  selector: 'caption[hlmCaption],caption[hlmTableCaption]',
  host: { 'data-slot': 'table-caption' },
})
export class HlmCaption {
  constructor() {
    classes(() => 'text-muted-foreground mt-4 text-sm');
  }
}

export const HlmTableImports = [
  HlmCaption,
  HlmTableContainer,
  HlmTable,
  HlmTBody,
  HlmTd,
  HlmTFoot,
  HlmTh,
  HlmTHead,
  HlmTr,
] as const;
