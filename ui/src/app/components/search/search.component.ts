import { Component, OnInit, signal, inject, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule, Router } from '@angular/router';
import { combineLatest } from 'rxjs';
import { ApiService, EventSummary } from '../../services/api.service';
import { StarredService } from '../../services/starred.service';
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
  selector: 'app-search',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './search.component.html',
  styleUrl: './search.component.css'
})
export class SearchComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private api = inject(ApiService);
  private starredService = inject(StarredService);
  private titleService = inject(Title);
  public linkService = inject(LinkService);
  public auth = inject(AuthService);
  private router = inject(Router);

  constructor() {
    effect(() => {
      const q = this.query();
      if (q) {
        this.titleService.setTitle(`Search: ${q}`);
      } else {
        this.titleService.setTitle('Search');
      }
    });
  }

  year = signal<number>(new Date().getFullYear());
  query = signal<string>('');
  orgId = signal<number | undefined>(undefined);
  events = signal<EventSummary[]>([]);
  loading = signal<boolean>(true);
  hideSoldOut = signal<boolean>(false);
  scrolled = signal<boolean>(false);
  isScrollingToAnchor = signal<boolean>(false);
  private scrollTimeout: any;
  collapsedGroups = signal<Set<string>>(new Set());
  groupingMethod = signal<'system' | 'year' | 'rating'>('system');

  // Advanced Search Filters
  filterFree = signal<boolean>(false);
  minTickets = signal<number | null>(null);
  minBggRating = signal<number | null>(null);
  minYearPublished = signal<number | null>(null);
  selectedDays = signal<Set<string>>(new Set(['wed', 'thu', 'fri', 'sat', 'sun']));
  selectedCategories = signal<Set<string>>(new Set());
  showAdvancedFilters = signal<boolean>(false);
  categorySearchQuery = signal<string>('');
  showCategoryDropdown = signal<boolean>(false);
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

  toggleCategory(catCode: string): void {
    const set = new Set(this.selectedCategories());
    if (set.has(catCode)) {
      set.delete(catCode);
    } else {
      set.add(catCode);
    }
    this.selectedCategories.set(set);
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
    this.selectedCategories.set(new Set());
    this.hideSoldOut.set(false);
    localStorage.removeItem('gcp_search_hideSoldOut');
    this.localSearchQuery.set('');
    this.updateQueryParams();
  }

  setMinTickets(event: Event): void {
    const input = event.target as HTMLInputElement;
    const val = input.value ? parseInt(input.value, 10) : null;
    this.minTickets.set(val !== null && val >= 0 ? val : null);
    this.updateQueryParams();
  }

  setMinBggRating(event: Event): void {
    const input = event.target as HTMLInputElement;
    const val = input.value ? parseFloat(input.value) : null;
    this.minBggRating.set(val !== null && val >= 0 ? val : null);
    this.updateQueryParams();
  }

  setMinYearPublished(event: Event): void {
    const input = event.target as HTMLInputElement;
    const val = input.value ? parseInt(input.value, 10) : null;
    this.minYearPublished.set(val !== null && val >= 0 ? val : null);
    this.updateQueryParams();
  }

  onCategorySearchInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.categorySearchQuery.set(input.value || '');
  }

  onCategorySearchBlur(): void {
    setTimeout(() => {
      this.showCategoryDropdown.set(false);
    }, 250);
  }

  selectCategory(code: string): void {
    this.toggleCategory(code);
    this.categorySearchQuery.set('');
  }

  focusCategoryInput(input: HTMLInputElement): void {
    input.focus();
  }

  onSearchQuerySubmit(): void {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {
        q: this.localSearchQuery() || null
      },
      queryParamsHandling: 'merge'
    });
  }

  filteredCategories = computed(() => {
    const query = this.categorySearchQuery().toLowerCase().trim();
    return this.categoriesList.filter(cat => 
      !this.selectedCategories().has(cat.code) &&
      (cat.name.toLowerCase().includes(query) || cat.code.toLowerCase().includes(query))
    );
  });

  activeFiltersCount = computed(() => {
    let count = 0;
    if (this.filterFree()) count++;
    if (this.minTickets() !== null && this.minTickets()! > 0) count++;
    if (this.minBggRating() !== null && this.minBggRating()! > 0) count++;
    if (this.minYearPublished() !== null && this.minYearPublished()! > 0) count++;
    if (this.selectedDays().size < 5) count++;
    if (this.selectedCategories().size > 0) count++;
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

    if (this.selectedCategories().size > 0) {
      localStorage.setItem('gcp_search_cats', Array.from(this.selectedCategories()).join(','));
    } else {
      localStorage.removeItem('gcp_search_cats');
    }

    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {
        free: this.filterFree() ? 'true' : null,
        minTickets: this.minTickets() !== null && this.minTickets()! > 0 ? this.minTickets() : null,
        minBgg: this.minBggRating() !== null && this.minBggRating()! > 0 ? this.minBggRating() : null,
        minYear: this.minYearPublished() !== null && this.minYearPublished()! > 0 ? this.minYearPublished() : null,
        days: this.selectedDays().size < 5 ? Array.from(this.selectedDays()).join(',') : null,
        cats: this.selectedCategories().size > 0 ? Array.from(this.selectedCategories()).join(',') : null
      },
      queryParamsHandling: 'merge'
    });
  }

  get categoriesList() {
    return Object.entries(this.categoryMap)
      .map(([code, name]) => ({ code, name }))
      .sort((a, b) => a.name.localeCompare(b.name));
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

  public categoryMap: { [key: string]: string } = {
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

  groupedEvents = computed(() => {
    let allEvents = this.events();

    // Apply Advanced Search Filters
    allEvents = allEvents.filter(e => {
      // 1. Restrict to specific categories
      if (this.selectedCategories().size > 0 && !this.selectedCategories().has(e.categoryCode)) {
        return false;
      }

      // 2. Restrict by days and minimum tickets
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

      // 3. Minimum BGG Rating
      const minBgg = this.minBggRating();
      if (minBgg !== null && minBgg > 0) {
        const rating = e.gameSystem.bggRating;
        if (rating === undefined || rating === null || rating < minBgg) {
          return false;
        }
      }

      // 4. Minimum Game Release Year
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
      let majorKey = this.categoryMap[event.categoryCode] || event.categoryCode;
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

  private getScrollOffset(): number {
    const offset = getComputedStyle(document.documentElement).getPropertyValue('--total-scroll-offset');
    return parseInt(offset, 10) || 115;
  }

  scrollToAnchor(id: string): void {
    const element = document.getElementById(id);
    if (element) {
      this.isScrollingToAnchor.set(true);
      this.scrolled.set(true); 
      
      setTimeout(() => {
        const offset = this.getScrollOffset();
        const elementPosition = element.getBoundingClientRect().top + window.scrollY;
        const offsetPosition = elementPosition - offset;

        window.scrollTo({
          top: offsetPosition,
          behavior: 'smooth'
        });
        
        if (id !== 'top') {
          history.pushState(null, '', window.location.pathname + window.location.search + '#' + id);
        } else {
          history.pushState(null, '', window.location.pathname + window.location.search);
        }

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
        // Adjust for the sticky header (approx 60-100px) + navbar (50px)
        const headerOffset = this.scrolled() ? 120 : 160;
        window.scrollBy(0, -headerOffset);
      }
    }, 0);
  }

  private initialLoad = true;

  ngOnInit(): void {
    window.addEventListener('scroll', () => {
      this.scrolled.set(window.scrollY > 50);
    });
    combineLatest([this.route.params, this.route.queryParams]).subscribe(([params, queryParams]) => {
      const grouping = params['grouping'];
      let newGrouping: 'system' | 'year' | 'rating' = 'system';
      if (grouping === 'by_year') {
        newGrouping = 'year';
      } else if (grouping === 'by_rating') {
        newGrouping = 'rating';
      }

      const newQuery = queryParams['q'] || '';
      const newYear = +(queryParams['year'] || new Date().getFullYear());
      const newOrgId = queryParams['org_id'] ? +queryParams['org_id'] : undefined;
      const queryParamFree = queryParams['free'];
      const queryParamMinTickets = queryParams['minTickets'];
      const queryParamMinBgg = queryParams['minBgg'];
      const queryParamMinYear = queryParams['minYear'];
      const queryParamDays = queryParams['days'];
      const queryParamCats = queryParams['cats'];

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

      let resolvedCats = new Set<string>();
      if (queryParamCats !== undefined) {
        if (queryParamCats) resolvedCats = new Set(queryParamCats.split(','));
      } else {
        const savedCats = localStorage.getItem('gcp_search_cats');
        if (savedCats !== null) {
          resolvedCats = new Set(savedCats.split(','));
          needUrlUpdate = true;
        }
      }

      const savedHideSoldOut = localStorage.getItem('gcp_search_hideSoldOut');
      if (savedHideSoldOut !== null && savedHideSoldOut === 'true') {
        this.hideSoldOut.set(true);
      }

      let needsFetch = this.initialLoad;
      if (this.query() !== newQuery || this.year() !== newYear || this.orgId() !== newOrgId || this.filterFree() !== newFree) {
        needsFetch = true;
      }

      this.filterFree.set(newFree);
      this.minTickets.set(newMinTickets);
      this.minBggRating.set(newMinBgg);
      this.minYearPublished.set(newMinYear);
      this.selectedDays.set(resolvedDays);
      this.selectedCategories.set(resolvedCats);
      this.localSearchQuery.set(newQuery);

      if (needUrlUpdate) {
        this.updateQueryParams();
      }


      if (newGrouping !== this.groupingMethod()) {
        this.groupingMethod.set(newGrouping);
        // If the grouping changed but the data didn't (needsFetch is false), 
        // we should still scroll to top for the new view.
        if (!needsFetch) {
          this.scrollToAnchor('top');
        }
      }

      if (needsFetch) {
        this.query.set(newQuery);
        this.year.set(newYear);
        this.orgId.set(newOrgId);
        this.initialLoad = false;
        this.fetchResults();
        this.starredService.fetchStarred(this.year());
      }
    });
  }

  fetchResults(): void {
    if (!this.query() && !this.orgId()) {
      this.events.set([]);
      this.loading.set(false);
      return;
    }
    this.loading.set(true);
    this.api.searchEvents({
      year: this.year(),
      search: this.query(),
      org_id: this.orgId(),
      free: this.filterFree()
    }).subscribe({
      next: (data) => {
        this.events.set(data);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error searching events', err);
        this.loading.set(false);
      }
    });
  }
}
