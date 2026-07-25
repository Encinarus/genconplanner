import { Component, signal, inject, computed, OnInit, HostListener, OnDestroy, ElementRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router, NavigationEnd } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { AuthService } from '../../services/auth.service';
import { LinkService } from '../../services/link.service';
import { PartyService } from '../../services/party.service';
import { filter } from 'rxjs/operators';

@Component({
  selector: 'app-navbar',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule],
  templateUrl: './navbar.component.html',
  styleUrl: './navbar.component.css'
})
export class NavbarComponent implements OnInit, OnDestroy {
  private router = inject(Router);
  private authService = inject(AuthService);
  public linkService = inject(LinkService);
  public partyService = inject(PartyService);
  private elementRef = inject(ElementRef);
  
  private lastScrollTop = 0;
  private isNavHidden = false;

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const navToggler = this.elementRef.nativeElement.querySelector('#navToggler');
    if (navToggler && navToggler.classList.contains('show')) {
      const clickedInsideNav = this.elementRef.nativeElement.contains(event.target as Node);
      // Close if clicking outside the navbar, or if clicking an actual navigation link inside the navbar
      const clickedLink = (event.target as HTMLElement).closest('a:not(.dropdown-toggle)');
      if (!clickedInsideNav || clickedLink) {
        this.closeMobileMenu();
      }
    }
  }

  private closeMobileMenu(): void {
    const navToggler = this.elementRef.nativeElement.querySelector('#navToggler');
    if (navToggler && navToggler.classList.contains('show')) {
      navToggler.classList.remove('show');
      const toggleBtn = this.elementRef.nativeElement.querySelector('.navbar-toggler');
      if (toggleBtn) {
        toggleBtn.setAttribute('aria-expanded', 'false');
        toggleBtn.classList.add('collapsed');
      }
    }
  }

  year = signal<number>(new Date().getFullYear());
  supportedYears = computed(() => {
    const currentYear = new Date().getFullYear();
    const years = [];
    for (let y = currentYear; y >= 2021; y--) {
      years.push(y);
    }
    return years;
  });
  displayName = this.authService.displayName;
  isAdmin = this.authService.isAdmin;
  searchQuery = signal<string>('');
  partyForCurrentYear = computed(() => {
    const currentYear = this.year();
    return this.partyService.parties().find(p => p.year === currentYear) || null;
  });

  ngOnInit() {
    this.syncYearFromUrl();
    this.router.events.pipe(
      filter(event => event instanceof NavigationEnd)
    ).subscribe(() => {
      this.syncYearFromUrl();
      this.showNav();
      this.closeMobileMenu();
    });
  }

  ngOnDestroy() {
    this.showNav();
  }

  @HostListener('window:scroll')
  onWindowScroll() {
    if (window.innerWidth >= 768) {
      if (this.isNavHidden) {
        this.showNav();
      }
      return;
    }

    const st = Math.max(0, window.pageYOffset || document.documentElement.scrollTop);
    const scrollDelta = st - this.lastScrollTop;

    if (st > 40 && scrollDelta > 5) {
      this.hideNav();
    } else if (scrollDelta < -5 || st <= 40) {
      this.showNav();
    }

    this.lastScrollTop = st;
  }

  @HostListener('window:resize')
  onWindowResize() {
    if (window.innerWidth >= 768 && this.isNavHidden) {
      this.showNav();
    }
  }

  private hideNav() {
    if (!this.isNavHidden) {
      document.body.classList.add('nav-hidden');
      this.isNavHidden = true;
    }
  }

  private showNav() {
    if (this.isNavHidden || document.body.classList.contains('nav-hidden')) {
      document.body.classList.remove('nav-hidden');
      this.isNavHidden = false;
    }
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

  async popupSignIn() {
    await this.authService.signIn();
    if (this.authService.user()) {
      const urlTree = this.router.parseUrl(this.router.url);
      const returnUrl = urlTree.queryParams['returnUrl'];
      if (returnUrl) {
        this.router.navigateByUrl(returnUrl);
      }
    }
  }

  async signOut() {
    await this.authService.signOut();
    // If on a protected route like /starred, redirect to home/categories
    if (this.router.url.includes('/starred/')) {
      this.router.navigate(this.linkService.getCategoryRouterLink(this.year(), ''));
    }
  }
}
