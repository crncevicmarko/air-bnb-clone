import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { HeaderComponent } from './header/header.component';
import { MainPageComponent } from './main-page/main-page.component';
import { LoginComponent } from './auth/login/login.component';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { HTTP_INTERCEPTORS, HttpClientModule } from '@angular/common/http';
import { RegisterComponent } from './auth/register/register.component';
import { AccommoAddComponent } from './accommodation/accommo-add/accommo-add.component';
import { ProfileComponent } from './profile/profile.component';
import { TokenInterceptor } from './auth/interceptor/token.interceptor';
import { NgxCaptchaModule } from 'ngx-captcha';
import { AccommoInfoComponent } from './accommodation/accommo-info/accommo-info.component';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
// import { MatDialogModule } from '@angular/material/dialog';
import { PasswordEmailRecoveryComponent } from './auth/password-email-recovery/password-email-recovery.component';
import { ResetPasswordComponent } from './auth/reset-password/reset-password.component';
import { VerifyEmailComponent } from './auth/verify-email/verify-email.component';
import { NgbModule } from '@ng-bootstrap/ng-bootstrap';
import { MyAccommoComponent } from './accommodation/my-accommo/my-accommo.component';
import { AccommoListComponent } from './accommodation/accommo-list/accommo-list.component';
import { ReservationsComponent } from './reservations/reservations.component';

//FA
import { FontAwesomeModule } from '@fortawesome/angular-fontawesome';
import { ToastrModule } from 'ngx-toastr';
import { NotificationComponent } from './notification/notification.component';

@NgModule({
  declarations: [
    AppComponent,
    HeaderComponent,
    MainPageComponent,
    LoginComponent,
    RegisterComponent,
    AccommoAddComponent,
    ProfileComponent,
    AccommoInfoComponent,
    PasswordEmailRecoveryComponent,
    ResetPasswordComponent,
    VerifyEmailComponent,
    MyAccommoComponent,
    AccommoListComponent,
    ReservationsComponent,
    NotificationComponent,
  ],
  imports: [
    BrowserModule,
    AppRoutingModule,
    FormsModule,
    ReactiveFormsModule,
    HttpClientModule,
    NgxCaptchaModule,
    BrowserAnimationsModule,
    NgbModule,
    FontAwesomeModule,
    BrowserAnimationsModule,
    ToastrModule.forRoot(),
    // MatDialogModule,
  ],
  providers: [
    { provide: HTTP_INTERCEPTORS, useClass: TokenInterceptor, multi: true },
  ],
  bootstrap: [AppComponent],
})
export class AppModule {}
