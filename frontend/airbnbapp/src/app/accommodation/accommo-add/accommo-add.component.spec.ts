import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AccommoAddComponent } from './accommo-add.component';

describe('AccommoAddComponent', () => {
  let component: AccommoAddComponent;
  let fixture: ComponentFixture<AccommoAddComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [AccommoAddComponent]
    });
    fixture = TestBed.createComponent(AccommoAddComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
