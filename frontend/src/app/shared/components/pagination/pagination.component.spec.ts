import { ComponentFixture, TestBed } from '@angular/core/testing';
import { PaginationComponent } from './pagination.component';
import { Component, signal } from '@angular/core';

@Component({
  standalone: true,
  imports: [PaginationComponent],
  template: `
    <app-pagination
      [page]="page()"
      [limit]="limit()"
      [totalCount]="totalCount()"
      (pageChange)="onPageChange($event)"
    />
  `,
})
class TestHostComponent {
  page = signal(1);
  limit = signal(10);
  totalCount = signal(25);
  lastEmittedPage: number | null = null;

  onPageChange(newPage: number) {
    this.lastEmittedPage = newPage;
    this.page.set(newPage);
  }
}

describe('PaginationComponent', () => {
  let fixture: ComponentFixture<TestHostComponent>;
  let hostComponent: TestHostComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TestHostComponent, PaginationComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TestHostComponent);
    hostComponent = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('calculates totalPages correctly from totalCount and limit', () => {
    // 25 items, limit 10 -> 3 total pages
    const textContent = fixture.nativeElement.textContent;
    expect(textContent).toContain('Halaman 1 dari 3');
  });

  it('disables prev button on page 1 and enables next button', () => {
    const buttons = fixture.nativeElement.querySelectorAll('button');
    const prevButton: HTMLButtonElement = buttons[0];
    const nextButton: HTMLButtonElement = buttons[1];

    expect(prevButton.disabled).toBe(true);
    expect(nextButton.disabled).toBe(false);
  });

  it('emits pageChange when next button is clicked', () => {
    const buttons = fixture.nativeElement.querySelectorAll('button');
    const nextButton: HTMLButtonElement = buttons[1];

    nextButton.click();
    fixture.detectChanges();

    expect(hostComponent.lastEmittedPage).toBe(2);
    expect(fixture.nativeElement.textContent).toContain('Halaman 2 dari 3');
  });

  it('disables next button on the last page', () => {
    hostComponent.page.set(3);
    fixture.detectChanges();

    const buttons = fixture.nativeElement.querySelectorAll('button');
    const prevButton: HTMLButtonElement = buttons[0];
    const nextButton: HTMLButtonElement = buttons[1];

    expect(prevButton.disabled).toBe(false);
    expect(nextButton.disabled).toBe(true);
  });

  it('emits pageChange when prev button is clicked from page 2', () => {
    hostComponent.page.set(2);
    fixture.detectChanges();

    const buttons = fixture.nativeElement.querySelectorAll('button');
    const prevButton: HTMLButtonElement = buttons[0];

    prevButton.click();
    fixture.detectChanges();

    expect(hostComponent.lastEmittedPage).toBe(1);
  });
});
