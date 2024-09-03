import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AccommoInfoComponent } from './accommo-info.component';

describe('AccommoInfoComponent', () => {
  let component: AccommoInfoComponent;
  let fixture: ComponentFixture<AccommoInfoComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [AccommoInfoComponent]
    });
    fixture = TestBed.createComponent(AccommoInfoComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
