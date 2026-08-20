import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormArray, FormControl, FormGroup, Validators } from '@angular/forms';
import { SoapAssessmentSectionComponent } from './soap-assessment-section.component';

describe('SoapAssessmentSectionComponent', () => {
  let component: SoapAssessmentSectionComponent;
  let fixture: ComponentFixture<SoapAssessmentSectionComponent>;
  let parentForm: FormGroup;
  let diagnosisArray: FormArray;

  beforeEach(async () => {
    diagnosisArray = new FormArray([
      new FormGroup({
        kodeIcd: new FormControl('J00'),
        deskripsi: new FormControl('Common cold', [Validators.required]),
      }),
    ]);
    parentForm = new FormGroup({
      diagnosis: diagnosisArray,
    });

    await TestBed.configureTestingModule({
      imports: [SoapAssessmentSectionComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(SoapAssessmentSectionComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('diagnosisArray', diagnosisArray);
    fixture.detectChanges();
  });

  it('should render diagnosis rows and emit addDiagnosis event', () => {
    const addSpy = vi.fn();
    component.addDiagnosis.subscribe(addSpy);

    component.onAdd();
    expect(addSpy).toHaveBeenCalled();
  });

  it('should emit removeDiagnosis event when remove button clicked', () => {
    const removeSpy = vi.fn();
    component.removeDiagnosis.subscribe(removeSpy);

    component.onRemove(0);
    expect(removeSpy).toHaveBeenCalledWith(0);
  });
});
