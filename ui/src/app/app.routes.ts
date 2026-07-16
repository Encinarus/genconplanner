import { Routes } from '@angular/router';
import { authGuard } from './guards/auth.guard';
import { adminGuard } from './guards/admin.guard';

export const routes: Routes = [
  { path: '', redirectTo: 'cat/2026', pathMatch: 'full' },
  { 
    path: 'cat/:year', 
    loadComponent: () => import('./components/category-list/category-list.component').then(m => m.CategoryListComponent) 
  },
  { path: 'cat/:year/:cat', redirectTo: 'cat/:year/:cat/by_system', pathMatch: 'full' },
  { 
    path: 'cat/:year/:cat/:grouping', 
    loadComponent: () => import('./components/category-detail/category-detail.component').then(m => m.CategoryDetailComponent) 
  },
  { 
    path: 'event/:eid', 
    loadComponent: () => import('./components/event-detail/event-detail.component').then(m => m.EventDetailComponent) 
  },
  { path: 'search', redirectTo: 'search/by_system', pathMatch: 'full' },
  { 
    path: 'search/:grouping', 
    loadComponent: () => import('./components/search/search.component').then(m => m.SearchComponent) 
  },
  { path: 'starred/:year', redirectTo: 'starred/:year/calendar', pathMatch: 'full' },
  { 
    path: 'starred/:year/:tab', 
    loadComponent: () => import('./components/starred/starred.component').then(m => m.StarredComponent),
    canActivate: [authGuard] 
  },
  { 
    path: 'about', 
    loadComponent: () => import('./components/about/about.component').then(m => m.AboutComponent) 
  },
  { 
    path: 'user', 
    loadComponent: () => import('./components/user/user.component').then(m => m.UserComponent),
    canActivate: [authGuard]
  },
  { path: 'party/:id', redirectTo: 'party/:id/events', pathMatch: 'full' },
  { 
    path: 'party/:id/:tab', 
    loadComponent: () => import('./components/party/party.component').then(m => m.PartyComponent),
    canActivate: [authGuard]
  },
  {
    path: 'admin/orgs',
    loadComponent: () => import('./components/admin-orgs/admin-orgs.component').then(m => m.AdminOrgsComponent),
    canActivate: [authGuard, adminGuard]
  },
  // Redirect any other routes to home for now
  { path: '**', redirectTo: '' }
];
