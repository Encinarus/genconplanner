import { inject } from '@angular/core';
import { Router, CanActivateFn } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { effect } from '@angular/core';

export const authGuard: CanActivateFn = (route, state) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (auth.user()) {
    return true;
  }

  // Redirect to home (which will redirect to cat/:year)
  router.navigate(['/']);
  return false;
};
