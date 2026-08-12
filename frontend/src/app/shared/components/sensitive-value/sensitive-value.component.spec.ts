import { Component } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { SensitiveValueComponent } from './sensitive-value.component';

@Component({
  standalone: true,
  imports: [ReactiveFormsModule, SensitiveValueComponent],
  template: `<app-sensitive-value mode="input" [formControl]="control" />`,
})
class TestHostComponent {
  control = new FormControl('secret123');
}

describe('SensitiveValueComponent', () => {
  let fixture: ComponentFixture<TestHostComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TestHostComponent, SensitiveValueComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(TestHostComponent);
    fixture.detectChanges();
  });

  it('should render input element in input mode with hidden password by default', () => {
    const inputEl = fixture.nativeElement.querySelector('input') as HTMLInputElement;
    expect(inputEl).toBeTruthy();
    expect(inputEl.type).toBe('password');
  });

  it('should toggle input type between password and text when eye icon button is clicked', () => {
    const btn = fixture.nativeElement.querySelector('button') as HTMLButtonElement;
    const inputEl = fixture.nativeElement.querySelector('input') as HTMLInputElement;

    expect(inputEl.type).toBe('password');
    btn.click();
    fixture.detectChanges();
    expect(inputEl.type).toBe('text');

    btn.click();
    fixture.detectChanges();
    expect(inputEl.type).toBe('password');
  });

  it('should mask NIK in display mode showing last 4 digits only when unrevealed', async () => {
    const displayFixture = TestBed.createComponent(SensitiveValueComponent);
    displayFixture.componentRef.setInput('mode', 'display');
    displayFixture.componentRef.setInput('displayValue', '3271234567890009');
    displayFixture.detectChanges();

    const text = displayFixture.nativeElement.textContent;
    expect(text).toContain('••••••••••••0009');

    const btn = displayFixture.nativeElement.querySelector('button') as HTMLButtonElement;
    btn.click();
    displayFixture.detectChanges();

    expect(displayFixture.nativeElement.textContent).toContain('3271234567890009');
  });
});
