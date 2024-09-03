import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { MainPageComponent } from './main-page/main-page.component';
import { LoginComponent } from './auth/login/login.component';
import { RegisterComponent } from './auth/register/register.component';
import { AccommoAddComponent } from './accommodation/accommo-add/accommo-add.component';
import { ResetPasswordComponent } from './auth/reset-password/reset-password.component';
import { ProfileComponent } from './profile/profile.component';
import { roleGuard } from './auth/guard/role.guard';
import { loginGuard } from './auth/guard/login.guard';
import { PasswordEmailRecoveryComponent } from './auth/password-email-recovery/password-email-recovery.component';
import { AccommoInfoComponent } from './accommodation/accommo-info/accommo-info.component';
import { VerifyEmailComponent } from './auth/verify-email/verify-email.component';
import { MyAccommoComponent } from './accommodation/my-accommo/my-accommo.component';
import { ReservationsComponent } from './reservations/reservations.component';
import { NotificationComponent } from './notification/notification.component';

const routes: Routes = [
  {
    path: '',
    component: MainPageComponent,
  },
  {
    path: 'sendEmail',
    component: PasswordEmailRecoveryComponent,
  },
  {
    path: 'login',
    component: LoginComponent,
  },
  {
    path: 'register',
    component: RegisterComponent,
  },
  {
    path: 'profile',
    component: ProfileComponent,
    canActivate: [loginGuard],
  },
  {
    path: 'accommodations/info/:id',
    component: AccommoInfoComponent,
  },
  {
    path: 'accommodations/myAccommodations',
    component: MyAccommoComponent,
    canActivate: [loginGuard, roleGuard],
  },
  {
    path: 'accommodations/create',
    component: AccommoAddComponent,
    canActivate: [loginGuard, roleGuard],
  },
  {
    path: 'reservations',
    component: ReservationsComponent,
  },
  {
    path: 'verify-email',
    component: VerifyEmailComponent,
  },
  { path: 'reset-password', component: ResetPasswordComponent },
  { path: 'notifications', component: NotificationComponent, canActivate: [loginGuard],},
  {
    path: '**',
    redirectTo: '',
    pathMatch: 'full',
  },

];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule],
})
export class AppRoutingModule {}
