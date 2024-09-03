import { Component, OnInit } from '@angular/core';
import { AuthService } from '../service/auth.service';
import { Router } from '@angular/router';
import { ProfServiceService } from '../service/prof.service.service';
import {
  AbstractControl,
  FormBuilder,
  FormGroup,
  Validators,
} from '@angular/forms';
import { ToastrService } from 'ngx-toastr';

@Component({
  selector: 'app-profile',
  templateUrl: './profile.component.html',
  styleUrls: ['./profile.component.css'],
})
export class ProfileComponent implements OnInit {
  // grades: any[] = [];
  resetForm: FormGroup;
  user = {
    username: '',
    email: '',
    firstName: '',
    lastName: '',
    city: '',
    country: '',
    streetName: '',
    streetNumber: '',
  };
  showOldPassword: boolean = false;
  showNewPassword: boolean = false;
  showConfirmPassword: boolean = false;
  averageGrade: number = 0.0;
  isHostFeatured: boolean = false;
  hostId: string = '';
  constructor(
    private authService: AuthService,
    private profService: ProfServiceService,
    private router: Router,
    private fb: FormBuilder,
    private toastr: ToastrService
  ) {
    this.resetForm = this.fb.group(
      {
        oldPassword: ['', [Validators.required]],
        newPassword: [
          '',
          [
            Validators.required,
            Validators.minLength(8),
            this.passwordValidator,
          ],
        ],
        confirmPassword: ['', [Validators.required]],
      },
      {
        validators: this.confirmPasswordValidator,
      }
    );
  }

  ngOnInit(): void {
    this.profService.getUserInfo().subscribe({
      next: (data) => {
        console.log('User info:', data);
        this.hostId = data.userId;
        this.user.username = data.username;
        this.user.email = data.email;
        this.user.firstName = data.firstname;
        this.user.lastName = data.lastname;
        this.user.city = data.location.city;
        this.user.country = data.location.country;
        this.user.streetName = data.location.streetName;
        this.user.streetNumber = data.location.streetNumber;
        // this.averageGrade = data.averageGrade;
        this.authService.getUserById(this.hostId).subscribe({
          next: (data) => {
            console.log('host:', data);
            this.averageGrade = data.averageGrade;
            this.isHostFeatured = data.isHostFeatured;
          },
          error: (err) => {
            this.toastr.error(err.error.message);
          },
        });
      },
      error: (err) => {
        this.toastr.error(err.error.message);
      },
    });

    // this.profService.getAllHostGrades().subscribe({
    //   next: (data) => {
    //     console.log(data);
    //     this.grades = data;
    //   },
    //   error: (err) => {
    //     alert(err.error.message);
    //   },
    // });
  }

  togglePasswordVisibility(number: number) {
    if (number == 1) {
      this.showOldPassword = !this.showOldPassword;
      this.resetForm.get('oldPassword')?.updateValueAndValidity();
    } else if (number == 2) {
      this.showNewPassword = !this.showNewPassword;
      this.resetForm.get('newPassword')?.updateValueAndValidity();
    } else {
      this.showConfirmPassword = !this.showConfirmPassword;
      this.resetForm.get('confirmPassword')?.updateValueAndValidity();
    }
  }

  submitForm() {
    this.profService.updateUserInfo(this.user).subscribe({
      next: (data) => {
        this.toastr.success('User successfully updated');
      },
      error: (err) => {
        this.toastr.error(err.error.message);
      },
    });
  }

  logout() {
    this.authService.logout();
    this.router.navigate(['']);
  }

  changePassword() {
    if (this.resetForm.valid) {
      console.log(
        this.resetForm.get('oldPassword')?.value,
        this.resetForm.get('newPassword')?.value,
        this.resetForm.get('confirmPassword')?.value
      );
      this.authService
        .changePasswod(
          this.resetForm.get('oldPassword')?.value,
          this.resetForm.get('newPassword')?.value,
          this.resetForm.get('confirmPassword')?.value
        )
        .subscribe({
          next: (data) => {
            console.log('Usli u changePassword');
            this.toastr.success('Password succesfully changed');
            this.authService.logout();
            this.router.navigate(['login']);
          },
          error: (err) => {
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

  deleteUser() {
    this.authService.deleteUser().subscribe({
      next: (data) => {
        console.log('Usli u deleteUser');
        this.toastr.success('User succesfully deleted');
        this.authService.logout();
        this.router.navigate(['login']);
      },
      error: (err) => {
        console.log(err.error.message);
        this.toastr.error(err.error.message);
      },
    });
  }

  confirmPasswordValidator(group: FormGroup) {
    const newPassword = group.get('newPassword')?.value;
    const confirmPassword = group.get('confirmPassword')?.value;

    return newPassword === confirmPassword ? null : { notSame: true };
  }
}
