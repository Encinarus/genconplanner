import { inject } from '@angular/core';
import { Router, CanActivateFn } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { toObservable } from '@angular/core/rxjs-interop';
import { filter, map, take } from 'rxjs/operators';

export const authGuard: CanActivateFn = (route, state) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  const getRedirectTree = () => {
    // Redirect to home (which will redirect to cat/:year) but save the URL they tried to hit
    return router.parseUrl(`/?returnUrl=${encodeURIComponent(state.url)}`);
  };

  // If auth state is already known, evaluate immediately
  if (auth.authLoaded()) {
    if (auth.user()) return true;
    return getRedirectTree();
  }

  // Otherwise wait for Firebase to initialize
  return toObservable(auth.authLoaded).pipe(
    filter(loaded => loaded),
    take(1),
    map(() => {
      if (auth.user()) return true;
      return getRedirectTree();
    })
  );
};
