import { Component, OnInit, AfterViewInit, OnDestroy, signal, inject, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule, Router } from '@angular/router';
import { combineLatest } from 'rxjs';
import { ApiService, EventSummary } from '../../services/api.service';
import { LinkService } from '../../services/link.service';
import { Title } from '@angular/platform-browser';
import { AuthService } from '../../services/auth.service';

interface EventSubGroup {
  systemName: string;
  events: EventSummary[];
  bggRating?: number;
  numBggRatings?: number;
  yearPublished?: number;
  bggId?: number;
}

interface GroupedEvents {
  minorName: string;
  subGroups: EventSubGroup[];
  isSoldOut: boolean;
  totalEventGroups: number;
}

interface MajorGroup {
  name: string;
  minorGroups: GroupedEvents[];
}

@Component({
  selector: 'app-category-detail',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './category-detail.component.html',
  styleUrl: './category-detail.component.css'
})
export class CategoryDetailComponent implements OnInit, AfterViewInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private api = inject(ApiService);
  private titleService = inject(Title);
  public linkService = inject(LinkService);

  constructor() {
    effect(() => {
      const name = this.categoryName();
      if (name) {
        this.titleService.setTitle(name);
      }
    });
  }

  private observer: IntersectionObserver | null = null;
  public isScrollingToAnchor = signal<boolean>(false);
  private scrollTimeout: any;
  private intersectingIds = new Set<string>();

  year = signal<number>(0);
  categoryCode = signal<string>('');
  events = signal<EventSummary[]>([]);
  loading = signal<boolean>(true);
  hideSoldOut = signal<boolean>(false);
  scrolled = signal<boolean>(false);
  collapsedGroups = signal<Set<string>>(new Set());
  groupingMethod = signal<'system' | 'year' | 'rating'>('system');
  public auth = inject(AuthService);
  private router = inject(Router);

  // Advanced Search Filters
  filterFree = signal<boolean>(false);
  minTickets = signal<number | null>(null);
  minBggRating = signal<number | null>(null);
  minYearPublished = signal<number | null>(null);
  selectedDays = signal<Set<string>>(new Set(['wed', 'thu', 'fri', 'sat', 'sun']));
  showAdvancedFilters = signal<boolean>(false);
  localSearchQuery = signal<string>('');

  toggleDay(day: string): void {
    const set = new Set(this.selectedDays());
    if (set.has(day)) {
      set.delete(day);
    } else {
      set.add(day);
    }
    this.selectedDays.set(set);
    this.updateQueryParams();
  }

  toggleFilterFree(): void {
    this.filterFree.set(!this.filterFree());
    this.updateQueryParams();
  }

  resetFilters(): void {
    this.filterFree.set(false);
    this.minTickets.set(null);
    this.minBggRating.set(null);
    this.minYearPublished.set(null);
    this.selectedDays.set(new Set(['wed', 'thu', 'fri', 'sat', 'sun']));
    this.hideSoldOut.set(false);
    localStorage.removeItem('gcp_search_hideSoldOut');
    this.localSearchQuery.set('');
    this.updateQueryParams();
  }

  setMinTickets(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    this.minTickets.set(val ? +val : null);
    this.updateQueryParams();
  }

  setMinBggRating(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    this.minBggRating.set(val ? +val : null);
    this.updateQueryParams();
  }

  setMinYearPublished(event: Event): void {
    const val = (event.target as HTMLInputElement).value;
    this.minYearPublished.set(val ? +val : null);
    this.updateQueryParams();
  }

  onSearchQuerySubmit(): void {
    if (this.localSearchQuery()) {
      this.router.navigate(['/search', `by_${this.groupingMethod()}`], {
        queryParams: {
          year: this.year(),
          cats: this.categoryCode(),
          free: this.filterFree() ? 'true' : null,
          minTickets: this.minTickets() !== null && this.minTickets()! > 0 ? this.minTickets() : null,
          minBgg: this.minBggRating() !== null && this.minBggRating()! > 0 ? this.minBggRating() : null,
          minYear: this.minYearPublished() !== null && this.minYearPublished()! > 0 ? this.minYearPublished() : null,
          days: this.selectedDays().size < 5 ? Array.from(this.selectedDays()).join(',') : null,
          q: this.localSearchQuery()
        }
      });
    } else {
      this.updateQueryParams();
    }
  }

  activeFiltersCount = computed(() => {
    let count = 0;
    if (this.filterFree()) count++;
    if (this.minTickets() !== null && this.minTickets()! > 0) count++;
    if (this.minBggRating() !== null && this.minBggRating()! > 0) count++;
    if (this.minYearPublished() !== null && this.minYearPublished()! > 0) count++;
    if (this.selectedDays().size < 5) count++;
    return count;
  });

  updateQueryParams(): void {
    // Write to localStorage
    if (this.filterFree()) {
      localStorage.setItem('gcp_search_free', 'true');
    } else {
      localStorage.removeItem('gcp_search_free');
    }

    if (this.minTickets() !== null && this.minTickets()! > 0) {
      localStorage.setItem('gcp_search_minTickets', this.minTickets()!.toString());
    } else {
      localStorage.removeItem('gcp_search_minTickets');
    }

    if (this.minBggRating() !== null && this.minBggRating()! > 0) {
      localStorage.setItem('gcp_search_minBgg', this.minBggRating()!.toString());
    } else {
      localStorage.removeItem('gcp_search_minBgg');
    }

    if (this.minYearPublished() !== null && this.minYearPublished()! > 0) {
      localStorage.setItem('gcp_search_minYear', this.minYearPublished()!.toString());
    } else {
      localStorage.removeItem('gcp_search_minYear');
    }

    if (this.selectedDays().size < 5) {
      localStorage.setItem('gcp_search_days', Array.from(this.selectedDays()).join(','));
    } else {
      localStorage.removeItem('gcp_search_days');
    }

    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {
        free: this.filterFree() ? 'true' : null,
        minTickets: this.minTickets() !== null && this.minTickets()! > 0 ? this.minTickets() : null,
        minBgg: this.minBggRating() !== null && this.minBggRating()! > 0 ? this.minBggRating() : null,
        minYear: this.minYearPublished() !== null && this.minYearPublished()! > 0 ? this.minYearPublished() : null,
        days: this.selectedDays().size < 5 ? Array.from(this.selectedDays()).join(',') : null,
        q: this.localSearchQuery() || null
      },
      queryParamsHandling: 'merge'
    });
  }

  setGrouping(method: 'system' | 'year' | 'rating'): void {
    this.router.navigate(['../by_' + method], { relativeTo: this.route, queryParamsHandling: 'preserve' });
  }

  toggleGroup(name: string): void {
    const set = new Set(this.collapsedGroups());
    if (set.has(name)) {
      set.delete(name);
    } else {
      set.add(name);
    }
    this.collapsedGroups.set(set);
  }
  
  isCollapsed(name: string): boolean {
    return this.collapsedGroups().has(name);
  }

  private categoryMap: { [key: string]: string } = {
    "ANI": "Anime Activities",
    "BGM": "Board Games",
    "CGM": "Non-Collectable/Tradable Card Games",
    "EGM": "Electronic Games",
    "ENT": "Entertainment Events",
    "ESC": "Escape Rooms",
    "FLM": "Film Fest",
    "HMN": "Historical Miniatures",
    "KID": "Kids Activities",
    "LRP": "Larps",
    "MHE": "Miniature Hobby Events",
    "NMN": "Non-Historical Miniatures",
    "RPG": "Role Playing Games",
    "RPGA": "Role Playing Game Association",
    "SEM": "Seminars",
    "SPA": "Supplemental Activities",
    "TCG": "Tradeable Card Game",
    "TDA": "True Dungeon",
    "TRD": "Trade Day Events",
    "WKS": "Workshop",
    "ZED": "Isle of Misfit Events"
  };

  categoryName = computed(() => this.categoryMap[this.categoryCode()] || this.categoryCode());

  groupedEvents = computed(() => {
    let allEvents = this.events();
    
    // Apply Advanced Search Filters
    allEvents = allEvents.filter(e => {
      // 1. Restrict by days and minimum tickets
      let minT = this.minTickets() || 0;
      if (this.hideSoldOut() && minT === 0) {
        minT = 1;
      }
      let matchesDayAndTickets = true;
      if (this.selectedDays().size > 0) {
        matchesDayAndTickets = Array.from(this.selectedDays()).some(day => {
          const tickets = day === 'wed' ? e.wedTickets :
                          day === 'thu' ? e.thuTickets :
                          day === 'fri' ? e.friTickets :
                          day === 'sat' ? e.satTickets :
                          day === 'sun' ? e.sunTickets : 0;
                          
          const events = day === 'wed' ? e.wedEvents :
                         day === 'thu' ? e.thuEvents :
                         day === 'fri' ? e.friEvents :
                         day === 'sat' ? e.satEvents :
                         day === 'sun' ? e.sunEvents : 0;
                         
          return events > 0 && tickets >= minT;
        });
      } else if (minT > 0) {
        matchesDayAndTickets = (e.wedEvents > 0 && e.wedTickets >= minT) || 
                               (e.thuEvents > 0 && e.thuTickets >= minT) || 
                               (e.friEvents > 0 && e.friTickets >= minT) || 
                               (e.satEvents > 0 && e.satTickets >= minT) || 
                               (e.sunEvents > 0 && e.sunTickets >= minT);
      }

      if (!matchesDayAndTickets) {
        return false;
      }

      // 2. Minimum BGG Rating
      const minBgg = this.minBggRating();
      if (minBgg !== null && minBgg > 0) {
        const rating = e.gameSystem.bggRating;
        if (rating === undefined || rating === null || rating < minBgg) {
          return false;
        }
      }

      // 3. Minimum Game Release Year
      const minYear = this.minYearPublished();
      if (minYear !== null && minYear > 0) {
        const year = e.gameSystem.yearPublished;
        if (year === undefined || year === null || year < minYear) {
          return false;
        }
      }

      return true;
    });

    if (allEvents.length === 0) return [];

    const majorGroupsMap = new Map<string, Map<string, Map<string, EventSummary[]>>>();

    allEvents.forEach(event => {
      let majorKey = this.categoryName();
      let minorKey = 'Unspecified';
      let subKey = event.gameSystem.name || 'Unspecified';
      
      if (this.groupingMethod() === 'system') {
        minorKey = subKey;
      } else if (this.groupingMethod() === 'year') {
        minorKey = event.gameSystem.yearPublished ? event.gameSystem.yearPublished.toString() : 'Unknown';
      } else if (this.groupingMethod() === 'rating') {
        minorKey = event.gameSystem.bggRating ? 'BGG ' + Math.floor(event.gameSystem.bggRating) : 'Unrated';
      }

      if (!majorGroupsMap.has(majorKey)) {
        majorGroupsMap.set(majorKey, new Map<string, Map<string, EventSummary[]>>());
      }
      const minorMap = majorGroupsMap.get(majorKey)!;
      if (!minorMap.has(minorKey)) {
        minorMap.set(minorKey, new Map<string, EventSummary[]>());
      }
      const subMap = minorMap.get(minorKey)!;
      if (!subMap.has(subKey)) {
        subMap.set(subKey, []);
      }
      subMap.get(subKey)!.push(event);
    });

    const result: MajorGroup[] = [];
    majorGroupsMap.forEach((minorMap, majorName) => {
      const minorGroups: GroupedEvents[] = [];
      minorMap.forEach((subMap, minorName) => {
        const subGroups: EventSubGroup[] = [];
        let minorTotalTickets = 0;
        let totalEventGroups = 0;
        
        subMap.forEach((events, systemName) => {
          events.sort((a, b) => a.title.localeCompare(b.title, undefined, { numeric: true, sensitivity: 'base' }));
          
          const subTotalTickets = events.reduce((sum, e) => sum + e.wedTickets + e.thuTickets + e.friTickets + e.satTickets + e.sunTickets, 0);
          minorTotalTickets += subTotalTickets;
          totalEventGroups += events.length;
          
          subGroups.push({
            systemName,
            events,
            bggRating: events[0].gameSystem.bggRating,
            numBggRatings: events[0].gameSystem.numBggRatings,
            yearPublished: events[0].gameSystem.yearPublished,
            bggId: events[0].gameSystem.bggId
          });
        });

        // Sort subgroups alphabetically
        subGroups.sort((a, b) => a.systemName.localeCompare(b.systemName, undefined, { numeric: true, sensitivity: 'base' }));

        minorGroups.push({
          minorName,
          subGroups,
          isSoldOut: minorTotalTickets === 0,
          totalEventGroups
        });
      });

      // Sort minor groups
      if (this.groupingMethod() === 'year') {
        minorGroups.sort((a, b) => {
          if (a.minorName === 'Unknown') return 1;
          if (b.minorName === 'Unknown') return -1;
          return b.minorName.localeCompare(a.minorName, undefined, { numeric: true, sensitivity: 'base' });
        });
      } else if (this.groupingMethod() === 'rating') {
        minorGroups.sort((a, b) => {
          if (a.minorName === 'Unrated') return 1;
          if (b.minorName === 'Unrated') return -1;
          return b.minorName.localeCompare(a.minorName, undefined, { numeric: true, sensitivity: 'base' });
        });
      } else {
        minorGroups.sort((a, b) => a.minorName.localeCompare(b.minorName, undefined, { numeric: true, sensitivity: 'base' }));
      }

      result.push({ name: majorName, minorGroups });
    });

    // Sort major groups by name
    result.sort((a, b) => a.name.localeCompare(b.name));

    return result;
  });

  filteredGroupsCount = computed(() => {
    return this.groupedEvents().reduce((acc, major) => {
      return acc + major.minorGroups.reduce((mSum, m) => {
        return mSum + m.subGroups.reduce((sSum, s) => sSum + s.events.length, 0);
      }, 0);
    }, 0);
  });

  totalSessionsCount = computed(() => {
    return this.groupedEvents().reduce((acc, major) => {
      return acc + major.minorGroups.reduce((mSum, m) => {
        return mSum + m.subGroups.reduce((sSum, s) => {
          return sSum + s.events.reduce((eSum, e) => eSum + e.numEvents, 0);
        }, 0);
      }, 0);
    }, 0);
  });

  toId(major: string, minor: string): string {
    return (major + '_' + minor).replace(/[^a-zA-Z0-9]/g, '_');
  }

  scrollToAnchor(id: string): void {
    const element = document.getElementById(id);
    if (element) {
      this.isScrollingToAnchor.set(true);
      
      // Force the header to its shrunken state immediately.
      this.scrolled.set(true); 

      // Wait for next tick so that the scrolled class is applied (without transition) and layout settles
      setTimeout(() => {
        const offset = this.getScrollOffset();
        const elementPosition = element.getBoundingClientRect().top + window.scrollY;
        const offsetPosition = elementPosition - offset;

        window.scrollTo({
          top: offsetPosition,
          behavior: 'smooth'
        });
        
        // Update hash immediately
        if (id === 'top') {
          history.pushState(null, '', window.location.pathname + window.location.search);
        } else {
          history.pushState(null, '', window.location.pathname + window.location.search + '#' + id);
        }

        // Reset the flag after animation finishes
        if (this.scrollTimeout) clearTimeout(this.scrollTimeout);
        this.scrollTimeout = setTimeout(() => {
          this.isScrollingToAnchor.set(false);
        }, 1500); 
      }, 0);
    }
  }

  toggleHideSoldOut(): void {
    const headings = Array.from(document.querySelectorAll('h5[id]'));
    let closestHeadingId = '';
    let minDiff = Number.MAX_VALUE;

    headings.forEach(h => {
      const rect = h.getBoundingClientRect();
      const diff = Math.abs(rect.top);
      if (diff < minDiff) {
        minDiff = diff;
        closestHeadingId = h.id;
      }
    });

    this.hideSoldOut.set(!this.hideSoldOut());
    if (this.hideSoldOut()) {
      localStorage.setItem('gcp_search_hideSoldOut', 'true');
    } else {
      localStorage.removeItem('gcp_search_hideSoldOut');
    }

    setTimeout(() => {
      const element = document.getElementById(closestHeadingId);
      if (element) {
        element.scrollIntoView({ behavior: 'auto', block: 'start' });
      }
    }, 0);
  }

  private initialLoad = true;

  ngOnInit(): void {
    window.addEventListener('scroll', this.onScroll);
    combineLatest([this.route.params, this.route.queryParams]).subscribe(([params, queryParams]) => {
      const newYear = +params['year'];
      const newCat = params['cat'];
      
      const queryParamFree = queryParams['free'];
      const queryParamMinTickets = queryParams['minTickets'];
      const queryParamMinBgg = queryParams['minBgg'];
      const queryParamMinYear = queryParams['minYear'];
      const queryParamDays = queryParams['days'];
      const queryParamQ = queryParams['q'] || '';

      let needUrlUpdate = false;

      let newFree = false;
      if (queryParamFree !== undefined) {
        newFree = queryParamFree === 'true';
      } else {
        const savedFree = localStorage.getItem('gcp_search_free');
        if (savedFree !== null) {
          newFree = savedFree === 'true';
          if (newFree) needUrlUpdate = true;
        }
      }

      let newMinTickets: number | null = null;
      if (queryParamMinTickets !== undefined) {
        newMinTickets = queryParamMinTickets ? +queryParamMinTickets : null;
      } else {
        const savedTickets = localStorage.getItem('gcp_search_minTickets');
        if (savedTickets !== null) {
          newMinTickets = +savedTickets;
          needUrlUpdate = true;
        }
      }

      let newMinBgg: number | null = null;
      if (queryParamMinBgg !== undefined) {
        newMinBgg = queryParamMinBgg ? +queryParamMinBgg : null;
      } else {
        const savedBgg = localStorage.getItem('gcp_search_minBgg');
        if (savedBgg !== null) {
          newMinBgg = +savedBgg;
          needUrlUpdate = true;
        }
      }

      let newMinYear: number | null = null;
      if (queryParamMinYear !== undefined) {
        newMinYear = queryParamMinYear ? +queryParamMinYear : null;
      } else {
        const savedYear = localStorage.getItem('gcp_search_minYear');
        if (savedYear !== null) {
          newMinYear = +savedYear;
          needUrlUpdate = true;
        }
      }

      let resolvedDays = new Set(['wed', 'thu', 'fri', 'sat', 'sun']);
      if (queryParamDays !== undefined) {
        if (queryParamDays) resolvedDays = new Set(queryParamDays.split(','));
      } else {
        const savedDays = localStorage.getItem('gcp_search_days');
        if (savedDays !== null) {
          resolvedDays = new Set(savedDays.split(','));
          needUrlUpdate = true;
        }
      }

      const savedHideSoldOut = localStorage.getItem('gcp_search_hideSoldOut');
      if (savedHideSoldOut !== null && savedHideSoldOut === 'true') {
        this.hideSoldOut.set(true);
      }

      let needsFetch = this.initialLoad;
      if (this.year() !== newYear || this.categoryCode() !== newCat || this.filterFree() !== newFree || this.localSearchQuery() !== queryParamQ) {
        this.year.set(newYear);
        this.categoryCode.set(newCat);
        needsFetch = true;
      }

      const grouping = params['grouping'];
      let newGrouping: 'system' | 'year' | 'rating' = 'system';
      if (grouping === 'by_year') {
        newGrouping = 'year';
      } else if (grouping === 'by_rating') {
        newGrouping = 'rating';
      }

      if (newGrouping !== this.groupingMethod()) {
        this.groupingMethod.set(newGrouping);
        // If the grouping changed but the data didn't (needsFetch is false), 
        // we should still scroll to top for the new view.
        if (!needsFetch) {
          this.scrollToAnchor('top');
        }
      }

      this.filterFree.set(newFree);
      this.minTickets.set(newMinTickets);
      this.minBggRating.set(newMinBgg);
      this.minYearPublished.set(newMinYear);
      this.selectedDays.set(resolvedDays);
      this.localSearchQuery.set(queryParamQ);

      if (needUrlUpdate) {
        this.updateQueryParams();
      }

      if (needsFetch) {
        this.initialLoad = false;
        this.fetchEvents();
      }
    });
  }

  private onScroll = () => {
    this.scrolled.set(window.scrollY > 50);
  }

  ngAfterViewInit(): void {
    this.setupIntersectionObserver();
  }

  ngOnDestroy(): void {
    window.removeEventListener('scroll', this.onScroll);
    if (this.observer) {
      this.observer.disconnect();
    }
    if (this.scrollTimeout) clearTimeout(this.scrollTimeout);
  }
  private getScrollOffset(): number {
    const offset = getComputedStyle(document.documentElement).getPropertyValue('--total-scroll-offset');
    return parseInt(offset, 10) || 115;
  }

  private setupIntersectionObserver(): void {
    // Only set up if we have events
    if (this.loading()) {
      // We might need to wait for events to load.
      // We can use an effect or just call this from fetchEvents.
      return;
    }

    if (this.observer) this.observer.disconnect();
    this.intersectingIds.clear();

    if (typeof IntersectionObserver === 'undefined') {
      return;
    }

    const offset = this.getScrollOffset();
    this.observer = new IntersectionObserver((entries) => {
      // ALWAYS update the state so it's not stale when we stop scrolling
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          this.intersectingIds.add(entry.target.id);
        } else {
          this.intersectingIds.delete(entry.target.id);
        }
      });

      // Skip URL updates while we are programmatically scrolling
      if (this.isScrollingToAnchor()) return;

      if (this.intersectingIds.size > 0) {
        // Find all headers and pick the first one that is currently intersecting our zone.
        const allHeaders = Array.from(document.querySelectorAll('h5[id]'));
        const firstVisible = allHeaders.find(h => this.intersectingIds.has(h.id));
        
        if (firstVisible && window.location.hash !== '#' + firstVisible.id) {
          history.replaceState(null, '', window.location.pathname + window.location.search + '#' + firstVisible.id);
        }
      }
    }, {
      // We use the exact offset for the rootMargin top.
      // We use a very narrow bottom margin so that we only consider the header "active" 
      // when it's right at the top of the content area.
      rootMargin: `-${offset}px 0px -95% 0px`, 
      threshold: 0
    });

    // Observe all h5 elements with IDs
    document.querySelectorAll('h5[id]').forEach(h => {
      this.observer?.observe(h);
    });
  }

  fetchEvents(): void {
    this.loading.set(true);
    this.api.searchEvents({
      year: this.year(),
      cat: this.categoryCode(),
      free: this.filterFree(),
      search: this.localSearchQuery() || undefined
    }).subscribe({
      next: (data) => {
        this.events.set(data);
        this.loading.set(false);
        
        // After view updates, handle initial hash and observer
        setTimeout(() => {
          this.setupIntersectionObserver();
          const hash = window.location.hash.substring(1);
          if (hash) {
            this.scrollToAnchor(hash);
          }
        }, 100);
      },
      error: (err) => {
        console.error('Error fetching events', err);
        this.loading.set(false);
      }
    });
  }
}
