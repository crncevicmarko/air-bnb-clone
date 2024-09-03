import { Component } from '@angular/core';
import {
  FormBuilder,
  FormGroup,
  Validators,
  AbstractControl,
} from '@angular/forms';
import { AuthService } from '../../service/auth.service';
import { group } from '@angular/animations';
import { Router } from '@angular/router';
import { ToastrService } from 'ngx-toastr';

@Component({
  selector: 'app-reset-password',
  templateUrl: './reset-password.component.html',
  styleUrls: ['./reset-password.component.css'],
})
export class ResetPasswordComponent {
  resetForm: FormGroup;

  constructor(
    private fb: FormBuilder,
    private router: Router,
    private authService: AuthService,
    private toastr: ToastrService
  ) {
    this.resetForm = this.fb.group(
      {
        newPassword: [
          '',
          [
            Validators.required,
            Validators.minLength(8),
            this.passwordValidator,
          ],
        ],
        confirmPassword: ['', [Validators.required]],
        secretCode: ['', [Validators.required, Validators.minLength(20)]],
      },
      {
        validators: this.confirmPasswordValidator,
      }
    );
  }

  resetPassword() {
    if (this.resetForm.valid) {
      this.authService
        .changeForgottenPassword(
          this.resetForm.get('newPassword')?.value,
          this.resetForm.get('confirmPassword')?.value,
          this.resetForm.get('secretCode')?.value
        )
        .subscribe({
          next: (data) => {
            console.log('Usli u resetPassword');
            this.toastr.success('Successfully changed password');
            this.router.navigate(['login']);
          },
          error: (err) => {
            console.log(err.error.message);
            this.toastr.error(err.error.message);
          },
        });
    } else {
      console.log('Form is invalid');
    }
  }

  passwordValidator(control: AbstractControl) {
    const passwordRegex =
      /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$/;
    return passwordRegex.test(control.value) ? null : { invalidPassword: true };
  }

  confirmPasswordValidator(group: FormGroup) {
    const newPassword = group.get('newPassword')?.value;
    const confirmPassword = group.get('confirmPassword')?.value;

    return newPassword === confirmPassword ? null : { notSame: true };
  }
}
