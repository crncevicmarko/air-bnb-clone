import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../../service/auth.service';
import { inject } from '@angular/core';
import { map } from 'rxjs';
import { ToastrService } from 'ngx-toastr';

export const roleGuard: CanActivateFn = (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);
  const toastr = inject(ToastrService)

  if (authService.getRole() !== 'HOST') {
    router.navigate(['']);
    toastr.warning("Only hosts can access this page")
    return false;
  } else return true;
};
