import { Component, OnInit, signal, inject, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, Event } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { StarredService } from '../../services/starred.service';
import { LinkService } from '../../services/link.service';
import { StarButtonComponent } from '../star-button/star-button.component';
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'app-event-detail',
  standalone: true,
  imports: [CommonModule, RouterModule, StarButtonComponent],
  templateUrl: './event-detail.component.html',
  styleUrl: './event-detail.component.css'
})
export class EventDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private api = inject(ApiService);
  private starredService = inject(StarredService);
  private titleService = inject(Title);
  public linkService = inject(LinkService);

  constructor() {
    effect(() => {
      const e = this.event();
      if (e) {
        this.titleService.setTitle(`Event: ${e.title}`);
      }
    });
  }

  eventId = signal<string>('');
  event = signal<Event | null>(null);
  loading = signal<boolean>(true);
  
  groupedEvents = computed(() => {
    const e = this.event();
    if (!e || !e.relatedEvents) return [];

    const groups: { [key: string]: any[] } = {};
    e.relatedEvents.forEach(rel => {
      // Use the raw date string to determine the day
      const date = new Date(rel.startTime);
      const day = date.toLocaleDateString('en-US', { weekday: 'long' });
      if (!groups[day]) {
        groups[day] = [];
      }
      groups[day].push(rel);
    });

    // Sort events within each day by time
    Object.keys(groups).forEach(day => {
      groups[day].sort((a, b) => new Date(a.startTime).getTime() - new Date(b.startTime).getTime());
    });

    // Sort days: Wed, Thu, Fri, Sat, Sun
    const dayOrder = ['Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
    return dayOrder
      .filter(day => groups[day])
      .map(day => ({
        day,
        events: groups[day]
      }));
  });

  allSessionIds = computed(() => {
    const e = this.event();
    if (!e || !e.relatedEvents) return [];
    return e.relatedEvents.map(rel => rel.eventId);
  });

  private auth = inject(AuthService);

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      this.eventId.set(params['eid']);
      this.fetchEvent();
    });
  }

  fetchEvent(): void {
    this.loading.set(true);
    this.api.getEvent(this.eventId()).subscribe({
      next: (data) => {
        this.event.set(data);
        this.starredService.fetchStarred(data.year);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error fetching event', err);
        this.loading.set(false);
      }
    });
  }

  isSessionStarred(sid: string): boolean {
    return this.starredService.isStarred(sid);
  }

  getEventTier(eventId: string): string {
    const data = this.starredService.starredPageData();
    if (!data) return '';
    const found = data.individualEvents.find(e => e.eventId === eventId);
    return found ? found.tier : '';
  }

  getGroupTier(eventId: string): string {
    const data = this.starredService.starredPageData();
    if (!data) return '';
    const found = data.individualEvents.find(e => e.eventId === eventId);
    return found ? (found.groupTier || '') : '';
  }

  isOverride(eventId: string): boolean {
    const data = this.starredService.starredPageData();
    if (!data) return false;
    const found = data.individualEvents.find(e => e.eventId === eventId);
    return found ? !!found.isOverride : false;
  }

  setGroupTier(eventId: string, year: number, tier: string): void {
    this.starredService.updateTier(eventId, year, tier, true);
  }

  handleGroupTierClick(eventId: string, year: number, clickedTier: string): void {
    if (this.getGroupTier(eventId) === clickedTier) {
      this.starredService.removeGroupDefault(eventId, year);
    } else {
      this.setGroupTier(eventId, year, clickedTier);
    }
  }

  setTier(eventId: string, year: number, tier: string): void {
    this.starredService.updateTier(eventId, year, tier, false);
  }

  resetOverride(eventId: string, year: number): void {
    this.starredService.removeOverride(eventId, year);
  }

  handleTierClick(eventId: string, year: number, clickedTier: string): void {
    if (this.isOverride(eventId) && this.getEventTier(eventId) === clickedTier) {
      this.resetOverride(eventId, year);
    } else if (!this.isOverride(eventId) && this.getEventTier(eventId) === clickedTier) {
      this.setTier(eventId, year, clickedTier);
    }
  }
}
