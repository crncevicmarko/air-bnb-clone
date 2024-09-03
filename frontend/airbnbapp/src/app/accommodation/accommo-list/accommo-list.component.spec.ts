import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AccommoListComponent } from './accommo-list.component';

describe('AccommoListComponent', () => {
  let component: AccommoListComponent;
  let fixture: ComponentFixture<AccommoListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [AccommoListComponent]
    });
    fixture = TestBed.createComponent(AccommoListComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
