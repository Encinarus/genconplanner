import { Component, signal, inject, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router, NavigationEnd } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { AuthService } from '../../services/auth.service';
import { LinkService } from '../../services/link.service';
import { filter } from 'rxjs/operators';

@Component({
  selector: 'app-navbar',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule],
  templateUrl: './navbar.component.html',
  styleUrl: './navbar.component.css'
})
export class NavbarComponent implements OnInit {
  private router = inject(Router);
  private authService = inject(AuthService);
  public linkService = inject(LinkService);
  
  year = signal<number>(new Date().getFullYear());
  supportedYears = computed(() => {
    const currentYear = new Date().getFullYear();
    const years = [];
    for (let y = currentYear; y >= 2021; y--) {
      years.push(y);
    }
    return years;
  });
  displayName = computed(() => this.authService.user()?.displayName || null);
  searchQuery = signal<string>('');

  ngOnInit() {
    this.syncYearFromUrl();
    this.router.events.pipe(
      filter(event => event instanceof NavigationEnd)
    ).subscribe(() => {
      this.syncYearFromUrl();
    });
  }

  private syncYearFromUrl() {
    const url = this.router.url;
    // Sync from path params (e.g., /starred/2026, /cat/2026)
    const pathMatch = url.match(/\/(?:starred|cat)\/(\d{4})/);
    if (pathMatch) {
      this.year.set(+pathMatch[1]);
      return;
    }
    
    // Sync from query params (e.g., ?year=2026)
    const urlTree = this.router.parseUrl(url);
    const yearParam = urlTree.queryParams['year'];
    if (yearParam) {
      this.year.set(+yearParam);
    }
  }

  setYear(newYear: number) {
    this.year.set(newYear);
    const urlTree = this.router.parseUrl(this.router.url);
    
    const segments = urlTree.root.children['primary']?.segments;
    
    // If at root or empty, go to the categories page for that year
    if (!segments || segments.length === 0) {
      this.router.navigate(this.linkService.getCategoryRouterLink(newYear, ''));
      return;
    }

    // Check if current path has a year segment
    if (segments.length >= 2) {
      const first = segments[0].path;
      if (first === 'starred' || first === 'cat') {
        // Replace the second segment (the year)
        segments[1].path = newYear.toString();
        this.router.navigateByUrl(urlTree.toString());
        return;
      }
    }

    // Otherwise, just update/add the year query param
    this.router.navigate([], { 
      queryParams: { year: newYear }, 
      queryParamsHandling: 'merge' 
    });
  }

  onSearch(event: Event) {
    event.preventDefault();
    if (this.searchQuery().trim()) {
      this.router.navigate(this.linkService.getSearchRouterLink(), { queryParams: { q: this.searchQuery(), year: this.year() } });
    }
  }

  popupSignIn() {
    this.authService.signIn();
  }

  async signOut() {
    await this.authService.signOut();
    // If on a protected route like /starred, redirect to home/categories
    if (this.router.url.includes('/starred/')) {
      this.router.navigate(this.linkService.getCategoryRouterLink(this.year(), ''));
    }
  }
}
