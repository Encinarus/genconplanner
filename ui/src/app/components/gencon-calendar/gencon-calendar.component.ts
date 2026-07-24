import { Component, Input, Output, EventEmitter, OnChanges, SimpleChanges, ViewChild, signal, ViewEncapsulation, ElementRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FullCalendarModule, FullCalendarComponent } from '@fullcalendar/angular';
import { CalendarOptions } from '@fullcalendar/core';
import dayGridPlugin from '@fullcalendar/daygrid';
import timeGridPlugin from '@fullcalendar/timegrid';
import bootstrap5Plugin from '@fullcalendar/bootstrap5';
import interactionPlugin from '@fullcalendar/interaction';
import listPlugin from '@fullcalendar/list';
import { getGenconDates } from '../../constants/gencon-dates';
import { estimateWalkTimeBetweenMapLinks } from '../../utils/walk-estimate';

declare var bootstrap: any;

export interface GenconCalendarEventItem {
  id: string;
  title: string;
  start: string; // ISO timestamp
  end: string;   // ISO timestamp
  url?: string;
  categoryCode?: string;
  location?: string;
  mapLink?: string;
  eventMapLink?: string;
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
  @Input() initialView: string = 'week';
  @Input() height: number | string = 525;
  @Input() showWalkEstimates: boolean = true;
  @Output() viewChange = new EventEmitter<string>();

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
          const eventId = props['eventId'] || arg.event.id;
          const location = props['location'];
          const mapLink = props['mapLink'];
          const holders: string[] = props['holderNames'] || [];
          const purchaserNames: string[] = props['purchaserNames'] || [];

          const mapUrl = mapLink || (location ? `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(location)}` : '');

          let locationHtml = '';
          if (location) {
            locationHtml = `
              <div class="small">
                <a href="${mapUrl || 'javascript:void(0)'}" target="_blank" rel="noopener noreferrer" class="text-primary text-decoration-underline location-link" onclick="event.stopPropagation()">
                  <i class="bi bi-geo-alt-fill me-1"></i>${location}
                </a>
              </div>
            `;
          }

          let html = `
            <div class="d-flex flex-column gap-1 py-1 fc-agenda-item">
              <div class="fw-bold fs-6 text-dark">
                ${cleanTitle}
              </div>
              ${eventId ? `<div class="small text-muted fw-medium">${eventId}</div>` : ''}
              ${locationHtml}
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
      const mapLink = props['mapLink'];
      const holders: string[] = props['holderNames'] || [];

      let html = `
        <div class="fc-event-main-frame" style="display: flex; flex-direction: column; justify-content: flex-start; gap: 1px; overflow: hidden; height: 100%; width: 100%; min-width: 0; max-width: 100%; box-sizing: border-box;">
          <div class="fc-event-title-line fw-bold" style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: block; width: 100%; min-width: 0; max-width: 100%; flex-shrink: 0; font-size: 0.8rem; line-height: 1.25; box-sizing: border-box;">${cleanTitle}</div>
          ${location ? (mapLink ? `<div class="fc-event-sub-line text-white-50" style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: block; width: 100%; min-width: 0; max-width: 100%; flex-shrink: 0; font-size: 0.72rem; line-height: 1.2; opacity: 0.85; box-sizing: border-box;"><a href="${mapLink}" target="_blank" rel="noopener noreferrer" style="color: inherit; text-decoration: underline;" onclick="event.stopPropagation()"><i class="bi bi-geo-alt-fill me-1"></i>${location}</a></div>` : `<div class="fc-event-sub-line text-white-50" style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: block; width: 100%; min-width: 0; max-width: 100%; flex-shrink: 0; font-size: 0.72rem; line-height: 1.2; opacity: 0.85; box-sizing: border-box;"><i class="bi bi-geo-alt-fill me-1"></i>${location}</div>`) : ''}
          ${holders.length > 0 ? `<div class="fc-event-sub-line text-white-50" style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: block; width: 100%; min-width: 0; max-width: 100%; flex-shrink: 0; font-size: 0.72rem; line-height: 1.2; opacity: 0.85; box-sizing: border-box;"><i class="bi bi-people-fill me-1"></i>${holders.join(', ')}</div>` : ''}
        </div>
      `;
      return { html };
    },
    scrollTime: '06:00:00',
    scrollTimeReset: false,
    height: 525,
    allDaySlot: false,
    editable: false,
    navLinks: false,
    nowIndicator: true,
    eventOrder: '-isMine,-rank',
    timeZone: 'America/Indiana/Indianapolis',
    eventClick: (info) => {
      const props = info.event.extendedProps;
      const location = props['location'];
      const mapLink = props['mapLink'];
      const mapUrl = mapLink || (location ? `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(location)}` : '');

      const targetUrl = (info.view.type === 'genconAgenda' && mapUrl) ? mapUrl : info.event.url;
      if (targetUrl) {
        info.jsEvent.preventDefault();
        window.open(targetUrl, '_blank');
      }
    },
    datesSet: (arg) => {
      const viewType = arg.view.type;
      const subView = viewType === 'genconAgenda' ? 'agenda' : viewType === 'timeGridDay' ? 'day' : 'week';
      this.viewChange.emit(subView);

      if (viewType === 'genconAgenda') {
        setTimeout(() => {
          this.scrollToCurrentOrNextAgendaEvent();
        }, 100);
      }
    },
    eventDidMount: (info) => {
      if (info.view.type === 'genconAgenda') {
        const catCode = info.event.extendedProps['categoryCode'];
        const color = info.event.backgroundColor || info.event.borderColor || this.getCategoryColor(catCode);
        if (color) {
          info.el.style.setProperty('--gencon-event-color', color);
        }

        const now = new Date();
        const nowMs = now.getTime();
        const startMs = info.event.start ? info.event.start.getTime() : 0;
        const endMs = info.event.end ? info.event.end.getTime() : startMs;

        if (endMs < nowMs) {
          info.el.classList.add('gencon-agenda-past');
        } else if ((startMs <= nowMs && nowMs <= endMs) || (startMs >= nowMs && startMs <= nowMs + 15 * 60 * 1000)) {
          info.el.classList.add('gencon-agenda-current');
        }

        if (this.showWalkEstimates) {
          const allEvents = info.view.calendar.getEvents().sort((a, b) => (a.start?.getTime() || 0) - (b.start?.getTime() || 0));
          const currentIndex = allEvents.findIndex(e => e.id === info.event.id || (e.start?.getTime() === startMs && e.title === info.event.title));
          if (currentIndex > 0) {
            const prevEvent = allEvents[currentIndex - 1];
            const prevEnd = prevEvent.end ? prevEvent.end.getTime() : (prevEvent.start ? prevEvent.start.getTime() : 0);
            const gapMs = startMs - prevEnd;

            if (gapMs >= 0 && gapMs < 60 * 60 * 1000) {
              const estimate = estimateWalkTimeBetweenMapLinks(
                prevEvent.extendedProps['mapLink'],
                info.event.extendedProps['mapLink']
              );
              if (estimate) {
                const parentNode = info.el.parentNode;
                const walkClass = `gencon-walk-row-${info.event.id}`;
                if (parentNode && !parentNode.querySelector(`.${walkClass}`)) {
                  const tr = document.createElement('tr');
                  tr.className = `gencon-walk-row ${walkClass}`;
                  tr.innerHTML = `
                    <td colspan="3" class="py-2 px-3 text-center bg-light border-top border-bottom text-muted small" style="background-color: #f8f9fa;">
                      <i class="bi bi-person-walking text-primary me-1"></i>
                      <span>Est. <strong>${estimate.displayText}</strong> between events</span>
                    </td>
                  `;
                  parentNode.insertBefore(tr, info.el);
                }
              }
            }
          }
        }
        return;
      }
      const props = info.event.extendedProps;
      const location = props['location'];
      const mapLink = props['mapLink'];
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
          ${location ? (mapLink ? `<div class="mb-1"><strong>Location:</strong> <a href="${mapLink}" target="_blank" rel="noopener noreferrer">${location}</a></div>` : `<div class="mb-1"><strong>Location:</strong> ${location}</div>`) : ''}
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
          mapLink: item.mapLink || item.eventMapLink || '',
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

    const fcInitialView = this.initialView === 'agenda' || this.initialView === 'genconAgenda' ? 'genconAgenda' : this.initialView === 'day' || this.initialView === 'timeGridDay' ? 'timeGridDay' : 'genconWeek';

    this.calendarOptions.update(opts => ({
      ...opts,
      height: this.height || 525,
      initialView: fcInitialView,
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
