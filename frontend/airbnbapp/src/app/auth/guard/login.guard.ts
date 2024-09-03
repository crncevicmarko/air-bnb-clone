import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../../service/auth.service';
import { inject } from '@angular/core';
import { map } from 'rxjs';
import { ToastrService } from 'ngx-toastr';

export const loginGuard: CanActivateFn = (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);
  const toastr = inject(ToastrService)

  if (authService.checkLoggin()) {
    return true;
  } else {
    router.navigate(['login']);
    toastr.warning("You have to be logged in to access that page")
    return false;
  }
};
