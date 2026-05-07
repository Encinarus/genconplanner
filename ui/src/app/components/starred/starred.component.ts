import { Component, OnInit, signal, inject, computed, effect, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, EventSummary, StarredEventDetail } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { StarredService } from '../../services/starred.service';
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
  imports: [CommonModule, RouterModule, FullCalendarModule],
  templateUrl: './starred.component.html',
  styleUrl: './starred.component.css'
})
export class StarredComponent implements OnInit {
  @ViewChild('calendar') calendarComponent!: FullCalendarComponent;

  private route = inject(ActivatedRoute);
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private starredService = inject(StarredService);
  private titleService = inject(Title);

  year = signal<number>(new Date().getFullYear());
  starredList = signal<StarredEventDetail[]>([]);
  loading = signal<boolean>(true);
  viewMode = signal<'list' | 'calendar'>('calendar');
  email = computed(() => this.auth.user()?.email || null);

  // Grouped and sorted starred events for the List view
  groupedStarredList = computed(() => {
    const groups: Record<string, StarredEventDetail[]> = {};
    this.starredList().forEach(e => {
      if (!groups[e.categoryCode]) groups[e.categoryCode] = [];
      groups[e.categoryCode].push(e);
    });
    
    return Object.entries(groups)
      .map(([code, events]) => ({ 
        code, 
        events: events.sort((a, b) => a.startTime.localeCompare(b.startTime))
      }))
      .sort((a, b) => a.code.localeCompare(b.code));
  });

  collapsedGroups = signal<Set<string>>(new Set());

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
        scrollTime: '06:00:00',
      }
    },
    height: 850,
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
    
    effect(() => {
      const year = this.year();
      const fallbackDate = `${year}-07-29`;
      this.calendarOptions.update(options => ({
        ...options,
        initialDate: fallbackDate
      }));
      
      // If auth is already loaded, re-fetch when year changes
      if (this.auth.authLoaded()) {
        this.fetchData();
      }
    });

    effect(() => {
      if (this.auth.authLoaded()) {
        this.fetchData();
      }
    });
  }

  fetchData(): void {
    const year = this.year();
    this.loading.set(true);

    this.api.getStarredPageData(year).subscribe({
      next: (data) => {
        // Update local state for list and calendar
        this.starredList.set(data.individualEvents);
        this.metadata = data.metadata;
        
        // Update calendar events
        const fcEvents = data.calendarEvents.map(e => ({
          title: e.title,
          start: e.startTime,
          end: e.endTime,
          url: e.plannerUrl,
          backgroundColor: this.categoryColors[e.shortCategory] || '#888888',
          borderColor: this.categoryColors[e.shortCategory] || '#888888',
          extendedProps: {
            description: e.shortDescription,
            similarCount: e.similarCount
          }
        }));

        this.hasWednesday = data.calendarEvents.some(e => {
          const d = new Date(e.startTime);
          return d.getDay() === 3;
        });

        const hiddenDays = this.hasWednesday ? [] : [3];
        let initialDate = this.getDesiredWeekStart();
        let duration = this.hasWednesday ? 5 : 4;

        const end = new Date(data.metadata.endDate);
        end.setDate(end.getDate() + 1);
        const inclusiveEndDate = end.toISOString().split('T')[0];

        this.calendarOptions.update(options => ({
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
          events: fcEvents
        }));

        // Sync global starred service state
        this.starredService.updateState(data.starredClusters || [], data.starredEvents || []);
        
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error fetching starred page data', err);
        this.loading.set(false);
      }
    });
  }

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      const newYear = +params['year'] || new Date().getFullYear();
      if (this.year() !== newYear) {
        this.year.set(newYear);
      }
      // No need to call starredService.fetchStarred here, 
      // as fetchData() will do it more efficiently
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

  setViewMode(mode: 'list' | 'calendar'): void {
    this.viewMode.set(mode);
  }

  toggleGroup(code: string): void {
    this.collapsedGroups.update(set => {
      const newSet = new Set(set);
      if (newSet.has(code)) newSet.delete(code);
      else newSet.add(code);
      return newSet;
    });
  }

  isCollapsed(code: string): boolean {
    return this.collapsedGroups().has(code);
  }

  formatTiming(start: string, end: string): string {
    const s = new Date(start);
    const e = new Date(end);
    const options: Intl.DateTimeFormatOptions = { 
      weekday: 'short', 
      hour: 'numeric', 
      minute: '2-digit',
      hour12: true 
    };
    return `${s.toLocaleDateString('en-US', options)} - ${e.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', hour12: true })}`;
  }
}
