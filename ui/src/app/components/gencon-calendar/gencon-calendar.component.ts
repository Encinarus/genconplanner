import { Component, Input, OnChanges, SimpleChanges, ViewChild, signal, ViewEncapsulation, ElementRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FullCalendarModule, FullCalendarComponent } from '@fullcalendar/angular';
import { CalendarOptions } from '@fullcalendar/core';
import dayGridPlugin from '@fullcalendar/daygrid';
import timeGridPlugin from '@fullcalendar/timegrid';
import bootstrap5Plugin from '@fullcalendar/bootstrap5';
import interactionPlugin from '@fullcalendar/interaction';
import listPlugin from '@fullcalendar/list';
import { getGenconDates } from '../../constants/gencon-dates';

declare var bootstrap: any;

export interface GenconCalendarEventItem {
  id: string;
  title: string;
  start: string; // ISO timestamp
  end: string;   // ISO timestamp
  url?: string;
  categoryCode?: string;
  location?: string;
  tier?: string;
  isMine?: boolean;
  rank?: number;
  status?: string;
  reasoning?: string[];
  holderNames?: string[];
  purchaserNames?: string[];
  description?: string;
  partyMembers?: { email: string; displayName: string; tier: string }[];
}

@Component({
  selector: 'app-gencon-calendar',
  standalone: true,
  imports: [CommonModule, FullCalendarModule],
  templateUrl: './gencon-calendar.component.html',
  styleUrl: './gencon-calendar.component.css',
  encapsulation: ViewEncapsulation.None
})
export class GenconCalendarComponent implements OnChanges {
  @ViewChild('calendar') calendarComponent!: FullCalendarComponent;
  @ViewChild('calendar', { read: ElementRef }) calendarElement!: ElementRef;

  @Input() events: GenconCalendarEventItem[] = [];
  @Input() year: number = new Date().getFullYear();
  @Input() userEmail: string = '';
  @Input() startDate?: string;
  @Input() endDate?: string;
  @Input() displayMode: string = 'all';

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

  getCategoryColor(code: string): string {
    if (!code) return '#0073AA';
    const clean = code.trim().toUpperCase();
    if (this.categoryColors[clean]) return this.categoryColors[clean];
    if (clean.length >= 4 && this.categoryColors[clean.substring(0, 4)]) {
      return this.categoryColors[clean.substring(0, 4)];
    }
    if (clean.length >= 3 && this.categoryColors[clean.substring(0, 3)]) {
      return this.categoryColors[clean.substring(0, 3)];
    }
    return '#0073AA';
  }

  calendarOptions = signal<CalendarOptions>({
    plugins: [dayGridPlugin, timeGridPlugin, bootstrap5Plugin, interactionPlugin, listPlugin],
    initialView: 'genconWeek',
    headerToolbar: {
      left: 'prev,next',
      center: 'title',
      right: 'timeGridDay,genconWeek,genconAgenda'
    },
    views: {
      genconWeek: {
        type: 'timeGrid',
        duration: { days: 5 },
        buttonText: 'week',
      },
      genconAgenda: {
        type: 'list',
        duration: { days: 5 },
        buttonText: 'agenda',
        eventContent: (arg) => {
          const props = arg.event.extendedProps;
          const cleanTitle = props['cleanTitle'] || arg.event.title;
          const location = props['location'];
          const holders: string[] = props['holderNames'] || [];
          const purchaserNames: string[] = props['purchaserNames'] || [];
          const startTimeStr = props['startTimeFormatted'] || '';

          let html = `
            <div class="d-flex flex-column gap-1 py-1 fc-agenda-item">
              <div class="fw-bold fs-6">
                <a href="${arg.event.url || 'javascript:void(0)'}" target="_blank" class="text-dark text-decoration-none">${cleanTitle}</a>
              </div>
              ${startTimeStr ? `<div class="small text-muted"><i class="bi bi-clock-fill me-1"></i>${startTimeStr}</div>` : ''}
              ${location ? `<div class="small text-muted"><i class="bi bi-geo-alt-fill me-1"></i>${location}</div>` : ''}
              ${holders.length > 0 ? `<div class="small text-dark"><i class="bi bi-people-fill me-1"></i><strong>Holders:</strong> ${holders.join(', ')}</div>` : ''}
              ${purchaserNames.length > 0 ? `<div class="small text-muted"><i class="bi bi-cart-check me-1"></i><strong>Purchased By:</strong> ${purchaserNames.join(', ')}</div>` : ''}
            </div>
          `;

          return { html };
        }
      }
    },
    eventContent: (arg) => {
      if (arg.view.type === 'genconAgenda') return null as any;
      const props = arg.event.extendedProps;
      const cleanTitle = props['cleanTitle'] || arg.event.title;
      const location = props['location'];
      const holders: string[] = props['holderNames'] || [];

      let html = `
        <div class="fc-event-main-frame" style="display: flex; flex-direction: column; justify-content: flex-start; gap: 2px; overflow: hidden; height: 100%; width: 100%; min-width: 0; max-width: 100%; box-sizing: border-box;">
          <div class="fc-event-title-line fw-bold" style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: block; width: 100%; min-width: 0; max-width: 100%; flex-shrink: 0; font-size: 0.82rem; line-height: 1.3; box-sizing: border-box;">${cleanTitle}</div>
          ${location ? `<div class="fc-event-sub-line text-white-50" style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: block; width: 100%; min-width: 0; max-width: 100%; flex-shrink: 1; font-size: 0.72rem; line-height: 1.25; opacity: 0.75; box-sizing: border-box;"><i class="bi bi-geo-alt-fill me-1"></i>${location}</div>` : ''}
          ${holders.length > 0 ? `<div class="fc-event-sub-line text-white-50" style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: block; width: 100%; min-width: 0; max-width: 100%; flex-shrink: 1; font-size: 0.72rem; line-height: 1.25; opacity: 0.75; box-sizing: border-box;"><i class="bi bi-people-fill me-1"></i>${holders.join(', ')}</div>` : ''}
        </div>
      `;
      return { html };
    },
    scrollTime: '06:00:00',
    scrollTimeReset: false,
    height: 850,
    allDaySlot: false,
    editable: false,
    navLinks: false,
    nowIndicator: true,
    eventOrder: '-isMine,-rank',
    timeZone: 'America/Indiana/Indianapolis',
    eventClick: (info) => {
      if (info.event.url) {
        info.jsEvent.preventDefault();
        window.open(info.event.url, '_blank');
      }
    },
    datesSet: (arg) => {
      if (arg.view.type === 'genconAgenda') {
        setTimeout(() => {
          this.scrollToCurrentOrNextAgendaEvent();
        }, 100);
      }
    },
    eventDidMount: (info) => {
      if (info.view.type === 'genconAgenda') {
        const now = new Date();
        const nowMs = now.getTime();
        const startMs = info.event.start ? info.event.start.getTime() : 0;
        const endMs = info.event.end ? info.event.end.getTime() : startMs;

        if (endMs < nowMs) {
          info.el.classList.add('gencon-agenda-past');
        } else if ((startMs <= nowMs && nowMs <= endMs) || (startMs >= nowMs && startMs <= nowMs + 15 * 60 * 1000)) {
          info.el.classList.add('gencon-agenda-current');
        }
        return;
      }
      const props = info.event.extendedProps;
      const location = props['location'];
      const cleanTitle = props['cleanTitle'] || info.event.title;
      const eventId = props['eventId'];
      const holders: string[] = props['holderNames'] || [];
      const purchaserNames: string[] = props['purchaserNames'] || [];
      const startTimeFormatted = props['startTimeFormatted'] || '';
      const description = props['description'] || '';
      const reasoning = (props['reasoning'] || []).join(', ');
      const status = props['status'];

      let content = `
        <div class="small">
          ${startTimeFormatted ? `<div class="mb-1"><strong>Time:</strong> ${startTimeFormatted}</div>` : ''}
          ${eventId ? `<div class="mb-1"><strong>Event ID:</strong> ${eventId}</div>` : ''}
          ${status ? `<div class="mb-1"><strong>Status:</strong> <span class="badge ${status === 'Primary' ? 'bg-success' : 'bg-secondary'}">${status}</span></div>` : ''}
          ${reasoning ? `<div class="mb-1"><strong>Reasoning:</strong> ${reasoning}</div>` : ''}
          ${location ? `<div class="mb-1"><strong>Location:</strong> ${location}</div>` : ''}
          ${holders.length > 0 ? `<div class="mb-1"><strong>Ticket Holders:</strong> ${holders.join(', ')}</div>` : ''}
          ${purchaserNames.length > 0 ? `<div class="mb-1"><strong>Purchasers:</strong> ${purchaserNames.join(', ')}</div>` : ''}
          ${description ? `<hr class="my-1"><div>${description}</div>` : ''}
        </div>
      `;

      info.el.setAttribute('data-bs-toggle', 'popover');
      info.el.setAttribute('data-bs-trigger', 'hover focus');
      info.el.setAttribute('title', cleanTitle);
      info.el.setAttribute('data-bs-content', content);

      if (typeof bootstrap !== 'undefined' && bootstrap.Popover) {
        new bootstrap.Popover(info.el, {
          html: true,
          container: 'body',
          trigger: 'hover focus',
          placement: 'auto'
        });
      }
    },
    eventWillUnmount: (info) => {
      if (typeof bootstrap !== 'undefined' && bootstrap.Popover) {
        const popover = bootstrap.Popover.getInstance(info.el);
        if (popover) {
          popover.dispose();
        }
      }
    }
  });

  scrollToCurrentOrNextAgendaEvent(): void {
    if (!this.calendarElement) return;
    const container = this.calendarElement.nativeElement;
    
    // First try: current or starting within 15 min
    let target: HTMLElement | null = container.querySelector('tr.fc-list-event.gencon-agenda-current');
    
    // Second try: next upcoming event (not past)
    if (!target) {
      const allEvents = Array.from(container.querySelectorAll('tr.fc-list-event')) as HTMLElement[];
      target = allEvents.find(el => !el.classList.contains('gencon-agenda-past')) || null;
    }
    
    if (target) {
      target.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['events'] || changes['year'] || changes['displayMode'] || changes['startDate']) {
      this.updateCalendar();
    }
  }

  getDesiredWeekStart(year: number): string {
    if (this.startDate) return this.startDate;
    return getGenconDates(year).startDate;
  }

  private updateCalendar(): void {
    const year = this.year || new Date().getFullYear();
    const wednesdayStart = this.getDesiredWeekStart(year);

    // Calculate boundary dates for Gen Con week (Wed .. Sun)
    const wedDate = new Date(wednesdayStart + 'T00:00:00Z');
    const thuDate = new Date(wedDate.getTime() + 24 * 3600 * 1000).toISOString().split('T')[0];
    const monDate = new Date(wedDate.getTime() + 5 * 24 * 3600 * 1000).toISOString().split('T')[0];

    const mode = this.displayMode || 'all';

    const rawEvents = this.events || [];
    const hasWednesday = rawEvents.some(e => {
      if (!e.start) return false;
      const d = new Date(e.start);
      return !isNaN(d.getTime()) && d.getDay() === 3;
    });

    const hiddenDays = hasWednesday ? [] : [3];
    const initialDate = this.startDate || (hasWednesday ? wednesdayStart : thuDate);
    const durationDays = hasWednesday ? 5 : 4;

    const filteredEvents = rawEvents.filter(item => {
      if (mode === 'only_mine' && !item.isMine) return false;
      if (mode === 'exclude_mine' && item.isMine) return false;
      return true;
    });

    const formattedEvents = filteredEvents.map(item => {
      const catCode = (item.categoryCode || (item.id && item.id.length >= 3 ? item.id.substring(0, 3) : '')).toUpperCase();
      const baseColor = this.getCategoryColor(catCode);

      const isMine = !!item.isMine;
      const isDimmed = mode === 'highlight_mine' && !isMine;

      const classNames: string[] = [];
      if (isDimmed) {
        classNames.push('dimmed-event');
      }

      let startTimeFormatted = '';
      if (item.start) {
        const dt = new Date(item.start);
        if (!isNaN(dt.getTime())) {
          startTimeFormatted = dt.toLocaleDateString('en-US', { weekday: 'short' }) + ' ' +
            dt.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
        }
      }

      const isMineValue = item.isMine !== undefined ? (item.isMine ? 1 : 0) : 0;

      return {
        id: item.id,
        title: item.title,
        start: item.start,
        end: item.end,
        url: item.url || `/event/${item.id}`,
        backgroundColor: baseColor,
        borderColor: baseColor,
        classNames: classNames,
        extendedProps: {
          cleanTitle: item.title,
          eventId: item.id,
          location: item.location || '',
          categoryCode: catCode,
          tier: item.tier || 'very_interested',
          isMine: isMineValue,
          rank: item.rank || 0,
          status: item.status,
          reasoning: item.reasoning,
          holderNames: item.holderNames || [],
          purchaserNames: item.purchaserNames || [],
          startTimeFormatted: startTimeFormatted,
          description: item.description || ''
        }
      };
    });

    this.calendarOptions.update(opts => ({
      ...opts,
      initialDate: initialDate,
      validRange: {
        start: wednesdayStart,
        end: monDate
      },
      hiddenDays: hiddenDays,
      views: {
        ...opts.views,
        genconWeek: {
          ...opts.views?.['genconWeek'],
          duration: { days: durationDays }
        },
        genconAgenda: {
          ...opts.views?.['genconAgenda'],
          duration: { days: durationDays }
        }
      },
      events: formattedEvents
    }));

    if (this.calendarComponent) {
      try {
        this.calendarComponent.getApi()?.gotoDate(initialDate);
      } catch {
        // Ignored if calendar view not yet rendered
      }
    }
  }
}
