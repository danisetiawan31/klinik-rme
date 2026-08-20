import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { SoapObjectiveSectionComponent } from './soap-objective-section.component';

describe('SoapObjectiveSectionComponent', () => {
  let component: SoapObjectiveSectionComponent;
  let fixture: ComponentFixture<SoapObjectiveSectionComponent>;
  let form: FormGroup;

  beforeEach(async () => {
    form = new FormGroup({
      hasilPemeriksaan: new FormControl('', [Validators.required, Validators.minLength(3)]),
    });

    await TestBed.configureTestingModule({
      imports: [SoapObjectiveSectionComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(SoapObjectiveSectionComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('form', form);
    fixture.detectChanges();
  });

  it('should render objective section header and textarea', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Pemeriksaan Fisik & Tanda Vital');
    expect(compiled.querySelector('textarea#hasilPemeriksaan')).toBeTruthy();
  });

  it('should show validation error when touched and invalid', () => {
    form.get('hasilPemeriksaan')?.setValue('ab');
    form.get('hasilPemeriksaan')?.markAsTouched();
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Hasil pemeriksaan fisik wajib diisi minimal 3 karakter');
  });
});
