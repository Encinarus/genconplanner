import { Component, OnInit, signal, inject, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, Event, PartyTicket } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { StarredService } from '../../services/starred.service';
import { LinkService } from '../../services/link.service';
import { PartyService } from '../../services/party.service';
import { Title } from '@angular/platform-browser';

import { TierSelectorComponent } from '../tier-selector/tier-selector.component';
import { BggLinkComponent } from '../bgg-link/bgg-link.component';

@Component({
  selector: 'app-event-detail',
  standalone: true,
  imports: [CommonModule, RouterModule, TierSelectorComponent, BggLinkComponent],
  templateUrl: './event-detail.component.html',
  styleUrl: './event-detail.component.css'
})
export class EventDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private api = inject(ApiService);
  private starredService = inject(StarredService);
  private titleService = inject(Title);
  public linkService = inject(LinkService);
  public partyService = inject(PartyService);
  public auth = inject(AuthService);

  // Filters
  filterHasTickets = signal<boolean>(typeof localStorage !== 'undefined' ? localStorage.getItem('filter_has_tickets') === 'true' : false);
  filterFreeTime = signal<boolean>(typeof localStorage !== 'undefined' ? localStorage.getItem('filter_free_time') === 'true' : false);
  filterPartyAvailable = signal<boolean>(typeof localStorage !== 'undefined' ? localStorage.getItem('filter_party_available') === 'true' : false);

  partyTickets = signal<PartyTicket[]>([]);

  currentParty = computed(() => {
    const e = this.event();
    if (!e) return null;
    return this.partyService.parties().find(p => p.year === e.year) || null;
  });

  isInParty = computed(() => {
    return this.currentParty() !== null;
  });

  constructor() {
    effect(() => {
      const e = this.event();
      if (e) {
        this.titleService.setTitle(`Event: ${e.title}`);
      }
    });

    effect(() => {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem('filter_has_tickets', this.filterHasTickets() ? 'true' : 'false');
      }
    });

    effect(() => {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem('filter_free_time', this.filterFreeTime() ? 'true' : 'false');
      }
    });

    effect(() => {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem('filter_party_available', this.filterPartyAvailable() ? 'true' : 'false');
      }
    });

    effect(() => {
      const party = this.currentParty();
      if (party) {
        this.api.getPartyTickets(party.year).subscribe({
          next: (res) => {
            this.partyTickets.set(res.tickets || []);
          },
          error: (err) => {
            console.error('Error fetching party tickets', err);
            this.partyTickets.set([]);
          }
        });
      } else {
        this.partyTickets.set([]);
      }
    });
  }

  eventId = signal<string>('');
  event = signal<Event | null>(null);
  loading = signal<boolean>(true);
  
  timeOverlaps(startA: string, endA: string, startB: string, endB: string): boolean {
    if (!startA || !endA || !startB || !endB) return false;
    return new Date(startA).getTime() < new Date(endB).getTime() && 
           new Date(startB).getTime() < new Date(endA).getTime();
  }

  getOtherAvailableMembersCount(session: any): number {
    const party = this.currentParty();
    if (!party) return 0;
    const currentUser = this.auth.user();
    const currentUserEmail = currentUser?.email?.toLowerCase();
    
    // Other party members
    const otherMembers = party.members.filter(m => m.email.toLowerCase() !== currentUserEmail);
    if (otherMembers.length === 0) return 0;

    // Active party tickets
    const tickets = this.partyTickets();

    let count = 0;
    otherMembers.forEach(member => {
      const memberEmail = member.email.toLowerCase();
      const hasOverlap = tickets.some(t => {
        if (t.ticketStatus === 'returned') return false;
        if (t.holderEmail.toLowerCase() !== memberEmail) return false;
        if (!t.eventStartTime || !t.eventEndTime) return false;
        return this.timeOverlaps(session.startTime, session.endTime, t.eventStartTime, t.eventEndTime);
      });
      if (!hasOverlap) {
        count++;
      }
    });

    return count;
  }

  getOtherAvailableMembersNames(session: any): string {
    const party = this.currentParty();
    if (!party) return '';
    const currentUser = this.auth.user();
    const currentUserEmail = currentUser?.email?.toLowerCase();
    
    // Other party members
    const otherMembers = party.members.filter(m => m.email.toLowerCase() !== currentUserEmail);
    if (otherMembers.length === 0) return '';

    // Active party tickets
    const tickets = this.partyTickets();

    const availableNames: string[] = [];
    otherMembers.forEach(member => {
      const memberEmail = member.email.toLowerCase();
      const hasOverlap = tickets.some(t => {
        if (t.ticketStatus === 'returned') return false;
        if (t.holderEmail.toLowerCase() !== memberEmail) return false;
        if (!t.eventStartTime || !t.eventEndTime) return false;
        return this.timeOverlaps(session.startTime, session.endTime, t.eventStartTime, t.eventEndTime);
      });
      if (!hasOverlap) {
        availableNames.push(member.displayName || member.email);
      }
    });

    return availableNames.join(', ');
  }

  getOtherAvailableMembers(session: any): any[] {
    const party = this.currentParty();
    if (!party) return [];
    const currentUser = this.auth.user();
    const currentUserEmail = currentUser?.email?.toLowerCase();
    
    // Other party members
    const otherMembers = party.members.filter(m => m.email && m.email.toLowerCase() !== currentUserEmail);
    if (otherMembers.length === 0) return [];

    // Active party tickets
    const tickets = this.partyTickets();

    const available: any[] = [];
    otherMembers.forEach(member => {
      const memberEmail = member.email.toLowerCase();
      const hasOverlap = tickets.some(t => {
        if (t.ticketStatus === 'returned') return false;
        if (t.holderEmail.toLowerCase() !== memberEmail) return false;
        if (!t.eventStartTime || !t.eventEndTime) return false;
        return this.timeOverlaps(session.startTime, session.endTime, t.eventStartTime, t.eventEndTime);
      });
      if (!hasOverlap) {
        available.push(member);
      }
    });

    return available;
  }

  sessionMatchesFilters(session: any): boolean {
    if (this.filterHasTickets() && session.ticketsAvailable === 0) {
      return false;
    }

    if (this.filterFreeTime()) {
      const myPurchased = this.starredService.starredPageData()?.individualEvents.filter(ev => ev.tier === 'purchased') || [];
      const overlaps = myPurchased.some(purchased => 
        this.timeOverlaps(session.startTime, session.endTime, purchased.startTime, purchased.endTime)
      );
      if (overlaps) {
        return false;
      }
    }

    return true;
  }

  hasActiveFilters = computed(() => {
    return this.filterHasTickets() || this.filterFreeTime();
  });

  matchedEvents = computed(() => {
    const e = this.event();
    if (!e || !e.relatedEvents) return [];
    return e.relatedEvents.filter(s => this.sessionMatchesFilters(s));
  });

  filteredOutEvents = computed(() => {
    const e = this.event();
    if (!e || !e.relatedEvents) return [];
    if (!this.hasActiveFilters()) return [];
    return e.relatedEvents.filter(s => !this.sessionMatchesFilters(s));
  });

  groupSessionsByDay(sessions: any[]): any[] {
    const groups: { [key: string]: any[] } = {};
    sessions.forEach(rel => {
      const date = new Date(rel.startTime);
      const day = date.toLocaleDateString('en-US', { weekday: 'long' });
      if (!groups[day]) {
        groups[day] = [];
      }
      groups[day].push(rel);
    });

    Object.keys(groups).forEach(day => {
      groups[day].sort((a, b) => new Date(a.startTime).getTime() - new Date(b.startTime).getTime());
    });

    const dayOrder = ['Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
    return dayOrder
      .filter(day => groups[day])
      .map(day => ({
        day,
        events: groups[day]
      }));
  }

  groupedMatchedEvents = computed(() => {
    return this.groupSessionsByDay(this.matchedEvents());
  });

  groupedFilteredOutEvents = computed(() => {
    return this.groupSessionsByDay(this.filteredOutEvents());
  });

  allSessionIds = computed(() => {
    const e = this.event();
    if (!e || !e.relatedEvents) return [];
    return e.relatedEvents.map(rel => rel.eventId);
  });

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
    } else {
      this.setTier(eventId, year, clickedTier);
    }
  }
}
