import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { SoapSubjectiveSectionComponent } from './soap-subjective-section.component';

describe('SoapSubjectiveSectionComponent', () => {
  let component: SoapSubjectiveSectionComponent;
  let fixture: ComponentFixture<SoapSubjectiveSectionComponent>;
  let form: FormGroup;

  beforeEach(async () => {
    form = new FormGroup({
      keluhan: new FormControl('', [Validators.required, Validators.minLength(3)]),
    });

    await TestBed.configureTestingModule({
      imports: [SoapSubjectiveSectionComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(SoapSubjectiveSectionComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('form', form);
    fixture.detectChanges();
  });

  it('should render subjective section header and textarea', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Anamnesis / Keluhan Pasien');
    expect(compiled.querySelector('textarea#keluhan')).toBeTruthy();
  });

  it('should show validation error when touched and invalid', () => {
    form.get('keluhan')?.setValue('a');
    form.get('keluhan')?.markAsTouched();
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Keluhan pasien wajib diisi minimal 3 karakter');
  });
});
