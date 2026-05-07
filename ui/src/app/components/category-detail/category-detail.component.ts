import { Component, OnInit, AfterViewInit, OnDestroy, signal, inject, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule, Router } from '@angular/router';
import { ApiService, EventSummary } from '../../services/api.service';
import { LinkService } from '../../services/link.service';
import { Title } from '@angular/platform-browser';

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
  private isScrollingToAnchor = false;
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
  private router = inject(Router);

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
    "SEM": "Seminiars",
    "SPA": "Spousal Activities",
    "TCG": "Tradeable Card Game",
    "TDA": "True Dungeon",
    "TRD": "Trade Day Events",
    "WKS": "Workshop",
    "ZED": "Isle of Misfit Events"
  };

  categoryName = computed(() => this.categoryMap[this.categoryCode()] || this.categoryCode());

  groupedEvents = computed(() => {
    let allEvents = this.events();
    if (this.hideSoldOut()) {
      allEvents = allEvents.filter(e => (e.wedTickets + e.thuTickets + e.friTickets + e.satTickets + e.sunTickets) > 0);
    }
    
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
      this.isScrollingToAnchor = true;
      
      // Force the header to its shrunken state immediately.
      // This ensures the browser's scroll calculation (which accounts for scroll-padding-top)
      // aligns correctly with the final header height after the transition.
      this.scrolled.set(true); 

      element.scrollIntoView({ behavior: 'smooth', block: 'start' });
      
      // Update hash immediately
      if (id === 'top') {
        history.pushState(null, '', window.location.pathname + window.location.search);
      } else {
        history.pushState(null, '', window.location.pathname + window.location.search + '#' + id);
      }

      // Reset the flag after animation finishes
      if (this.scrollTimeout) clearTimeout(this.scrollTimeout);
      this.scrollTimeout = setTimeout(() => {
        this.isScrollingToAnchor = false;
      }, 1500); // Increased timeout to ensure smooth scroll completes
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
    this.route.params.subscribe(params => {
      const newYear = +params['year'];
      const newCat = params['cat'];
      
      let needsFetch = this.initialLoad;
      if (this.year() !== newYear || this.categoryCode() !== newCat) {
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
      if (this.isScrollingToAnchor) return;

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
    this.api.searchEvents({ year: this.year(), cat: this.categoryCode() }).subscribe({
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
