import { Component, OnInit, signal, inject, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule, Router } from '@angular/router';
import { combineLatest } from 'rxjs';
import { ApiService, EventSummary } from '../../services/api.service';
import { StarredService } from '../../services/starred.service';
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

  groupedEvents = computed(() => {
    let allEvents = this.events();
    if (this.hideSoldOut()) {
      allEvents = allEvents.filter(e => (e.wedTickets + e.thuTickets + e.friTickets + e.satTickets + e.sunTickets) > 0);
    }

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

      let needsFetch = this.initialLoad;
      if (this.query() !== newQuery || this.year() !== newYear || this.orgId() !== newOrgId) {
        needsFetch = true;
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
      org_id: this.orgId()
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
