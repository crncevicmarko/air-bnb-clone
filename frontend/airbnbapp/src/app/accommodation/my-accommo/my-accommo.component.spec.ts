import { ComponentFixture, TestBed } from '@angular/core/testing';

import { MyAccommoComponent } from './my-accommo.component';

describe('MyAccommoComponent', () => {
  let component: MyAccommoComponent;
  let fixture: ComponentFixture<MyAccommoComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [MyAccommoComponent]
    });
    fixture = TestBed.createComponent(MyAccommoComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
