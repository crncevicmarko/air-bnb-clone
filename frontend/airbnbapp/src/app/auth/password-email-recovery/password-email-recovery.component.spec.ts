import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PasswordEmailRecoveryComponent } from './password-email-recovery.component';

describe('PasswordEmailRecoveryComponent', () => {
  let component: PasswordEmailRecoveryComponent;
  let fixture: ComponentFixture<PasswordEmailRecoveryComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [PasswordEmailRecoveryComponent]
    });
    fixture = TestBed.createComponent(PasswordEmailRecoveryComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
