import { Component } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../service/auth.service';
import { ToastrService } from 'ngx-toastr';

@Component({
  selector: 'app-verify-email',
  templateUrl: './verify-email.component.html',
  styleUrls: ['./verify-email.component.css'],
})
export class VerifyEmailComponent {
  verifyEmail: FormGroup;
  constructor(
    private fb: FormBuilder,
    private router: Router,
    private authService: AuthService,
    private toastr: ToastrService
  ) {
    this.verifyEmail = this.fb.group({
      code: ['', [Validators.required]],
    });
  }
  sendVerificationCode() {
    if (this.verifyEmail.valid) {
      const code = this.verifyEmail.get('code')?.value;
      //
      this.authService.sendVerifyingEmail(code).subscribe({
        next: (data) => {
          console.log('Email sent successfully');
          this.toastr.success('Email verified successfully.');
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
}
