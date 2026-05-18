import { HttpInterceptorFn } from '@angular/common/http';
import Cookies from 'js-cookie';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const token = Cookies.get('signinToken');
  if (token) {
    const cloned = req.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
    return next(cloned);
  }
  return next(req);
};
