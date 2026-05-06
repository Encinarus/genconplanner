import { Routes } from '@angular/router';
import { CategoryListComponent } from './components/category-list/category-list.component';
import { AboutComponent } from './components/about/about.component';
import { CategoryDetailComponent } from './components/category-detail/category-detail.component';
import { EventDetailComponent } from './components/event-detail/event-detail.component';
import { SearchComponent } from './components/search/search.component';
import { StarredComponent } from './components/starred/starred.component';

export const routes: Routes = [
  { path: '', redirectTo: 'cat/2026', pathMatch: 'full' },
  { path: 'cat/:year', component: CategoryListComponent },
  { path: 'cat/:year/:cat', redirectTo: 'cat/:year/:cat/by_system', pathMatch: 'full' },
  { path: 'cat/:year/:cat/:grouping', component: CategoryDetailComponent },
  { path: 'event/:eid', component: EventDetailComponent },
  { path: 'search', redirectTo: 'search/by_system', pathMatch: 'full' },
  { path: 'search/:grouping', component: SearchComponent },
  { path: 'starred/:year', component: StarredComponent },
  { path: 'about', component: AboutComponent },
  // Redirect any other routes to home for now
  { path: '**', redirectTo: '' }
];
