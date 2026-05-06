import { Component, OnInit, signal, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, EventSummary } from '../../services/api.service';
import { StarredService } from '../../services/starred.service';
import { StarButtonComponent } from '../star-button/star-button.component';

interface GroupedEvents {
  minorName: string;
  events: EventSummary[];
  bggRating?: number;
  yearPublished?: number;
  isSoldOut: boolean;
  bggId?: number;
}

interface MajorGroup {
  name: string;
  minorGroups: GroupedEvents[];
}

@Component({
  selector: 'app-search',
  standalone: true,
  imports: [CommonModule, RouterModule, StarButtonComponent],
  templateUrl: './search.component.html',
  styleUrl: './search.component.css'
})
export class SearchComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private api = inject(ApiService);
  private starredService = inject(StarredService);

  year = signal<number>(new Date().getFullYear());
  query = signal<string>('');
  orgId = signal<number | undefined>(undefined);
  events = signal<EventSummary[]>([]);
  loading = signal<boolean>(true);
  hideSoldOut = signal<boolean>(false);
  scrolled = signal<boolean>(false);

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

    const majorGroupsMap = new Map<string, Map<string, EventSummary[]>>();

    allEvents.forEach(event => {
      let majorKey = this.categoryMap[event.categoryCode] || event.categoryCode;
      let minorKey = event.gameSystem.name || 'Unspecified';

      if (!majorGroupsMap.has(majorKey)) {
        majorGroupsMap.set(majorKey, new Map<string, EventSummary[]>());
      }
      const minorMap = majorGroupsMap.get(majorKey)!;
      if (!minorMap.has(minorKey)) {
        minorMap.set(minorKey, []);
      }
      minorMap.get(minorKey)!.push(event);
    });

    const result: MajorGroup[] = [];
    majorGroupsMap.forEach((minorMap, majorName) => {
      const minorGroups: GroupedEvents[] = [];
      minorMap.forEach((events, minorName) => {
        events.sort((a, b) => a.title.localeCompare(b.title, undefined, { numeric: true, sensitivity: 'base' }));
        const totalTickets = events.reduce((sum, e) => sum + e.wedTickets + e.thuTickets + e.friTickets + e.satTickets + e.sunTickets, 0);

        minorGroups.push({
          minorName,
          events,
          bggRating: events[0].gameSystem.bggRating,
          yearPublished: events[0].gameSystem.yearPublished,
          isSoldOut: totalTickets === 0,
          bggId: events[0].gameSystem.bggId
        });
      });
      minorGroups.sort((a, b) => a.minorName.localeCompare(b.minorName, undefined, { numeric: true, sensitivity: 'base' }));
      result.push({ name: majorName, minorGroups });
    });

    result.sort((a, b) => a.name.localeCompare(b.name));

    return result;
  });

  filteredGroupsCount = computed(() => {
    return this.groupedEvents().reduce((acc, major) => {
      return acc + major.minorGroups.reduce((mSum, m) => mSum + m.events.length, 0);
    }, 0);
  });

  totalSessionsCount = computed(() => {
    return this.groupedEvents().reduce((acc, major) => {
      return acc + major.minorGroups.reduce((mSum, m) => {
        return mSum + m.events.reduce((eSum, e) => eSum + e.numEvents, 0);
      }, 0);
    }, 0);
  });

  toId(major: string, minor: string): string {
    return (major + '_' + minor).replace(/[^a-zA-Z0-9]/g, '_');
  }

  scrollToAnchor(id: string): void {
    const element = document.getElementById(id);
    if (element) {
      element.scrollIntoView({ behavior: 'smooth' });
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

  ngOnInit(): void {
    window.addEventListener('scroll', () => {
      this.scrolled.set(window.scrollY > 50);
    });
    this.route.queryParams.subscribe(params => {
      this.query.set(params['q'] || '');
      this.year.set(+(params['year'] || new Date().getFullYear()));
      this.orgId.set(params['org_id'] ? +params['org_id'] : undefined);
      this.fetchResults();
      this.starredService.fetchStarred(this.year());
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
