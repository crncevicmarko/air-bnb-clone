import { Component } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../service/auth.service';
import { ToastrService } from 'ngx-toastr';

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.css'],
})
export class LoginComponent {
  form: FormGroup;
  siteKey: string;

  constructor(
    private fb: FormBuilder,
    private router: Router,
    private authService: AuthService,
    private toastr: ToastrService
  ) {
    this.form = this.fb.group({
      username: [null, Validators.required],
      password: [null, Validators.required],
    });
    this.siteKey = '6LddmB4pAAAAALdViM1b2M9OJZNgwKQ-HbFtGXK-';
  }

  submit() {
    this.authService.login(this.form.value).subscribe({
      next: (data) => {
        console.log('login success');
        localStorage.setItem('jwt', data.access_token);
        this.router.navigate(['']);
        this.authService.checkLoggin();
        this.authService.checkRole();
        this.toastr.success("Successfully logged in!")
      },
      error: (err) => {
        console.log(err.error.message);
        this.toastr.error(err.error.message);
      },
    });
  }
}
