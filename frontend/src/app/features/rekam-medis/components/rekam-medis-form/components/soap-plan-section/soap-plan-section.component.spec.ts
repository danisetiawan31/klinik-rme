import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormArray, FormControl, FormGroup, Validators } from '@angular/forms';
import { SoapPlanSectionComponent } from './soap-plan-section.component';

describe('SoapPlanSectionComponent', () => {
  let component: SoapPlanSectionComponent;
  let fixture: ComponentFixture<SoapPlanSectionComponent>;
  let parentForm: FormGroup;
  let tindakanArray: FormArray;

  beforeEach(async () => {
    tindakanArray = new FormArray<FormGroup>([]);
    parentForm = new FormGroup({
      tindakan: tindakanArray,
    });

    await TestBed.configureTestingModule({
      imports: [SoapPlanSectionComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(SoapPlanSectionComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('tindakanArray', tindakanArray);
    fixture.detectChanges();
  });

  it('should render empty state when tindakanArray is empty', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Belum ada resep obat atau tindakan medis');
  });

  it('should emit addTindakan event with correct type', () => {
    const addSpy = vi.fn();
    component.addTindakan.subscribe(addSpy);

    component.onAdd('resep');
    expect(addSpy).toHaveBeenCalledWith('resep');

    component.onAdd('tindakan');
    expect(addSpy).toHaveBeenCalledWith('tindakan');
  });

  it('should render dynamic items when tindakanArray has items and emit remove event', () => {
    const testArray = new FormArray([
      new FormGroup({
        jenis: new FormControl('resep', [Validators.required]),
        deskripsi: new FormControl('Paracetamol 500mg', [Validators.required]),
      }),
    ]);
    fixture.componentRef.setInput('tindakanArray', testArray);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('select')).toBeTruthy();

    const removeSpy = vi.fn();
    component.removeTindakan.subscribe(removeSpy);

    component.onRemove(0);
    expect(removeSpy).toHaveBeenCalledWith(0);
  });
});
