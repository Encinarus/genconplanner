import { Component, OnInit, signal, inject, computed, effect, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterModule, Router } from '@angular/router';
import { ApiService, EventSummary, StarredEventDetail, StarredPageData } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { StarredService } from '../../services/starred.service';
import { LinkService } from '../../services/link.service';
import { Title } from '@angular/platform-browser';
import { FullCalendarModule, FullCalendarComponent } from '@fullcalendar/angular';
import { CalendarOptions } from '@fullcalendar/core';
import dayGridPlugin from '@fullcalendar/daygrid';
import timeGridPlugin from '@fullcalendar/timegrid';
import bootstrap5Plugin from '@fullcalendar/bootstrap5';
import interactionPlugin from '@fullcalendar/interaction';
import { forkJoin } from 'rxjs';

declare var bootstrap: any;

@Component({
  selector: 'app-starred',
  standalone: true,
  imports: [CommonModule, RouterModule, FullCalendarModule, FormsModule],
  templateUrl: './starred.component.html',
  styleUrl: './starred.component.css'
})
export class StarredComponent implements OnInit {
  @ViewChild('calendar') calendarComponent!: FullCalendarComponent;

  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private starredService = inject(StarredService);
  public linkService = inject(LinkService);
  private titleService = inject(Title);

  year = signal<number>(new Date().getFullYear());
  starredList = signal<StarredEventDetail[]>([]);
  loading = signal<boolean>(true);
  viewMode = signal<'list' | 'calendar' | 'bulk'>('calendar');
  tierFilter = signal<string>('all');
  bulkInput = signal<string>('');
  importAsGroups = signal<boolean>(false);
  email = computed(() => this.auth.user()?.email || null);

  // Grouped and sorted starred events for the List view
  groupedStarredList = computed(() => {
    const filter = this.tierFilter();
    const categoryGroups: Record<string, Record<string, StarredEventDetail[]>> = {};
    
    this.starredList().forEach(e => {
      // Apply tier filter
      const eventTier = e.tier || 'very_interested';
      if (filter !== 'all' && eventTier !== filter) return;

      if (!categoryGroups[e.categoryCode]) categoryGroups[e.categoryCode] = {};
      const groupKey = `${e.title}|${e.shortDescription}`;
      if (!categoryGroups[e.categoryCode][groupKey]) categoryGroups[e.categoryCode][groupKey] = [];
      categoryGroups[e.categoryCode][groupKey].push(e);
    });
    
    return Object.entries(categoryGroups)
      .map(([code, groupMap]) => ({ 
        code, 
        eventGroups: Object.entries(groupMap).map(([key, events]) => ({
            key,
            title: events[0].title,
            shortDescription: events[0].shortDescription,
            events: events.sort((a, b) => a.startTime.localeCompare(b.startTime))
        })).sort((a, b) => a.title.localeCompare(b.title))
      }))
      .sort((a, b) => a.code.localeCompare(b.code));
  });

  collapsedCategories = signal<Set<string>>(new Set());
  collapsedEventGroups = signal<Set<string>>(new Set());

  // Store metadata to help with view jumps
  private metadata: { startDate: string, endDate: string } | null = null;
  private hasWednesday: boolean = false;

  calendarOptions = signal<CalendarOptions>({
    plugins: [dayGridPlugin, timeGridPlugin, bootstrap5Plugin, interactionPlugin],
    initialView: 'genconWeek',
    headerToolbar: {
      left: 'prev,next',
      center: 'title',
      right: 'timeGridDay,genconWeek'
    },
    views: {
      genconWeek: {
        type: 'timeGrid',
        duration: { days: 5 },
        buttonText: 'week',
      }
    },
    scrollTime: '06:00:00',
    scrollTimeReset: false,
    height: 850,
    allDaySlot: false,
    editable: false,
    navLinks: false,
    nowIndicator: true,
    timeZone: 'America/Indiana/Indianapolis',
    eventClick: (info) => {
      if (info.event.url) {
        info.jsEvent.preventDefault();
        window.open(info.event.url, '_blank');
      }
    },
    eventDidMount: (info) => {
      const description = info.event.extendedProps['description'];
      if (description) {
        info.el.setAttribute('data-bs-toggle', 'popover');
        info.el.setAttribute('data-bs-trigger', 'hover focus');
        info.el.setAttribute('title', info.event.title);
        info.el.setAttribute('data-bs-content', description);
        
        if (typeof bootstrap !== 'undefined' && bootstrap.Popover) {
          new bootstrap.Popover(info.el, {
            html: true,
            container: 'body',
            trigger: 'hover focus',
            placement: 'auto'
          });
        }
      }
    },
    eventWillUnmount: (info) => {
      if (typeof bootstrap !== 'undefined' && bootstrap.Popover) {
        const popover = bootstrap.Popover.getInstance(info.el);
        if (popover) {
          popover.dispose();
        }
      }
    },
    datesSet: (info) => {
      if (info.view.type === 'genconWeek' && this.metadata) {
        const expectedStart = this.getDesiredWeekStart();
        if (info.startStr.split('T')[0] !== expectedStart) {
          setTimeout(() => {
            if (this.calendarComponent) {
              this.calendarComponent.getApi().gotoDate(expectedStart);
            }
          });
        }
      }
    }
  });

  private categoryColors: Record<string, string> = {
    'ANI': '#A9177E',
    'BGM': '#0073AA',
    'CGM': '#6B2355',
    'EGM': '#858E95',
    'ENT': '#C94088',
    'ESC': '#df4426',
    'FLM': '#4B4761',
    'HMN': '#2A3181',
    'KID': '#9470AA',
    'LRP': '#AE8B1C',
    'MHE': '#E8B51C',
    'NMN': '#686F1F',
    'RPG': '#448A80',
    'RPGA': '#D67917',
    'SEM': '#009CDF',
    'SPA': '#A6C749',
    'TCG': '#1C944A',
    'TDA': '#771F17',
    'TRD': '#878F68',
    'WKS': '#5E3C03',
    'ZED': '#75B9B8',
  };

  constructor() {
    this.titleService.setTitle('Starred Events');
    
    // React to data changes from the service (cache or API)
    effect(() => {
      const data = this.starredService.starredPageData();
      const currentYear = this.year();
      
      if (data && data.year === currentYear) {
        this.processData(data);
        this.loading.set(false);
      } else {
        // Only show loading if we don't have data for the current year yet
        this.loading.set(true);
      }
    });

    effect(() => {
      const year = this.year();
      const fallbackDate = `${year}-07-29`;
      this.calendarOptions.update(options => ({
        ...options,
        initialDate: fallbackDate
      }));
      
      if (this.auth.authLoaded()) {
        this.starredService.fetchStarred(year);
      }
    });
  }

  processData(data: StarredPageData): void {
    // Update local state for list and calendar
    this.starredList.set(data.individualEvents || []);
    this.metadata = data.metadata;
    
    // Client-side clustering
    const filter = this.tierFilter();
    
    // 1. Filter the raw sessions
    const filteredSessions = (data.individualEvents || []).filter(e => {
        const eventTier = e.tier || 'very_interested';
        return filter === 'all' || eventTier === filter;
    });

    // 2. Group by "Game" (Title + Description + Category) to maintain original clustering behavior
    const gameGroups: Record<string, StarredEventDetail[]> = {};
    filteredSessions.forEach(e => {
        const key = `${e.title}|${e.shortDescription}|${e.categoryCode}`;
        if (!gameGroups[key]) gameGroups[key] = [];
        gameGroups[key].push(e);
    });

    // 3. Cluster each game's sessions based on time overlaps
    const allClusters: any[] = [];
    Object.values(gameGroups).forEach(sessions => {
        allClusters.push(...this.clusterEvents(sessions));
    });

    this.hasWednesday = (data.individualEvents || []).some((e: any) => {
      const d = new Date(e.startTime);
      return d.getDay() === 3;
    });

    const hiddenDays = this.hasWednesday ? [] : [3];
    let initialDate = this.getDesiredWeekStart();
    let duration = this.hasWednesday ? 5 : 4;

    const end = new Date(data.metadata.endDate);
    end.setDate(end.getDate() + 1);
    const inclusiveEndDate = end.toISOString().split('T')[0];

    this.calendarOptions.update(options => {
      const hasStructureChanged = 
        options.initialDate !== initialDate ||
        options.hiddenDays?.length !== hiddenDays.length;

      if (hasStructureChanged) {
        return {
          ...options,
          initialDate: initialDate,
          validRange: {
            start: data.metadata.startDate,
            end: inclusiveEndDate
          },
          hiddenDays: hiddenDays,
          views: {
            ...options.views,
            genconWeek: {
              ...options.views?.['genconWeek'],
              duration: { days: duration }
            }
          },
          events: allClusters
        };
      } else {
        return {
          ...options,
          events: allClusters
        };
      }
    });
  }

  fetchData(): void {
    // This is now handled reactively, but we keep the method signature 
    // if other parts of the component depend on it, just redirected to the service.
    this.starredService.fetchStarred(this.year());
  }

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      const newYear = +params['year'] || new Date().getFullYear();
      if (this.year() !== newYear) {
        this.year.set(newYear);
      }

      const tab = params['tab'];
      if (tab && ['calendar', 'list', 'bulk'].includes(tab)) {
        this.viewMode.set(tab as any);
      }
    });
  }

  private getDesiredWeekStart(): string {
    if (!this.metadata) return '';
    if (this.hasWednesday) {
      return this.metadata.startDate;
    } else {
      const start = new Date(this.metadata.startDate);
      start.setDate(start.getDate() + 1);
      return start.toISOString().split('T')[0];
    }
  }

  setViewMode(mode: 'list' | 'calendar' | 'bulk'): void {
    this.router.navigate(['/starred', this.year(), mode]);
  }

  updateTier(eventId: string, tier: string): void {
    this.starredService.updateTier(eventId, this.year(), tier);
  }

  setTierFilter(tier: string): void {
    this.tierFilter.set(tier);
    // Re-process data to refresh calendar (and trigger computed list refresh)
    const data = this.starredService.starredPageData();
    if (data) {
      this.processData(data);
    }
  }

  onClearAll(): void {
    const year = this.year();
    if (confirm(`Are you sure you want to clear all starred events for ${year}?`)) {
      this.starredService.bulkClear(year).subscribe({
        next: () => {
          this.viewMode.set('list');
        },
        error: (err) => {
          alert('Error clearing events: ' + (err.error?.error || err.message));
        }
      });
    }
  }

  onBulkUpdate(overwrite: boolean): void {
    const year = this.year();
    const text = this.bulkInput();
    const yearLastTwo = (year % 100).toString().padStart(2, '0');
    
    // Regex for GenCon IDs: [A-Z]{3,4}YYND\d{6,}
    const idRegex = new RegExp(`[A-Z]{3,4}${yearLastTwo}ND\\d{6,}`, 'g');
    const matches: string[] = text.match(idRegex) || [];

    if (matches.length === 0) {
      alert(`No valid event IDs found for the year ${year}.`);
      return;
    }

    // Check for any IDs that match the general pattern but have a DIFFERENT year
    const generalIdRegex = /[A-Z]{3,4}\d{2}ND\d{6,}/g;
    const allMatches: string[] = text.match(generalIdRegex) || [];
    const wrongYearMatches = allMatches.filter(id => !matches.includes(id));

    if (wrongYearMatches.length > 0) {
      alert(`Input contains event IDs from a different year: ${wrongYearMatches.join(', ')}. Please only include IDs for ${year}.`);
      return;
    }

    const uniqueIds = [...new Set(matches)];
    let confirmMsg = `Identify ${uniqueIds.length} unique events for ${year}. `;
    if (overwrite) {
      confirmMsg += "This will FULLY REPLACE your current starred events. Are you sure?";
    } else {
      confirmMsg += "This will add these events to your current starred events. Proceed?";
    }

    if (confirm(confirmMsg)) {
      this.starredService.bulkReplace(year, text, overwrite, this.importAsGroups()).subscribe({
        next: () => {
          this.bulkInput.set('');
          this.viewMode.set('list');
        },
        error: (err) => {
          alert('Error updating events: ' + (err.error?.error || err.message));
        }
      });
    }
  }

  toggleCategory(code: string): void {
    this.collapsedCategories.update(set => {
      const newSet = new Set(set);
      if (newSet.has(code)) newSet.delete(code);
      else newSet.add(code);
      return newSet;
    });
  }

  isCategoryCollapsed(code: string): boolean {
    return this.collapsedCategories().has(code);
  }

  toggleEventGroup(key: string): void {
    this.collapsedEventGroups.update(set => {
      const newSet = new Set(set);
      if (newSet.has(key)) newSet.delete(key);
      else newSet.add(key);
      return newSet;
    });
  }

  isEventGroupCollapsed(key: string): boolean {
    return this.collapsedEventGroups().has(key);
  }

  unstarEvent(eventId: string): void {
    if (confirm('Are you sure you want to unstar this event session?')) {
        this.starredService.toggleStar(eventId, this.year(), false);
    }
  }

  formatTiming(start: string, end: string): string {
    const s = new Date(start);
    const e = new Date(end);
    const timeZone = 'America/Indiana/Indianapolis';
    const options: Intl.DateTimeFormatOptions = { 
      timeZone,
      weekday: 'short', 
      hour: 'numeric', 
      minute: '2-digit',
      hour12: true 
    };
    const timeOptions: Intl.DateTimeFormatOptions = {
      timeZone,
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    };
    return `${s.toLocaleDateString('en-US', options)} - ${e.toLocaleTimeString('en-US', timeOptions)}`;
  }

  private clusterEvents(events: StarredEventDetail[]): any[] {
    if (events.length === 0) return [];

    // Sort by start time
    const sorted = [...events].sort((a, b) => a.startTime.localeCompare(b.startTime));

    const clusters: any[] = [];
    let currentCluster: any = null;

    for (const event of sorted) {
      if (!currentCluster || event.startTime > currentCluster.end) {
        if (currentCluster) {
          if (currentCluster.extendedProps.similarCount > 1) {
            currentCluster.title = `${currentCluster.title}\n\n(${currentCluster.extendedProps.similarCount} similar)`;
          }
          clusters.push(currentCluster);
        }
        currentCluster = {
          title: event.title,
          start: event.startTime,
          end: event.endTime,
          url: this.linkService.getEventUrl(event.eventId),
          backgroundColor: this.categoryColors[event.categoryCode] || '#888888',
          borderColor: this.categoryColors[event.categoryCode] || '#888888',
          extendedProps: {
            description: event.shortDescription,
            similarCount: 1,
            tier: event.tier || 'very_interested'
          }
        };
      } else {
        if (event.endTime > currentCluster.end) {
          currentCluster.end = event.endTime;
        }
        currentCluster.extendedProps.similarCount++;
      }
    }

    if (currentCluster) {
      if (currentCluster.extendedProps.similarCount > 1) {
        currentCluster.title = `${currentCluster.title}\n\n(${currentCluster.extendedProps.similarCount} similar)`;
      }
      clusters.push(currentCluster);
    }

    return clusters;
  }
}
