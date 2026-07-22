import { TestBed, ComponentFixture } from '@angular/core/testing';
import { signal } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { EventDetailComponent } from './event-detail.component';
import { ApiService, Event } from '../../services/api.service';
import { StarredService } from '../../services/starred.service';
import { LinkService } from '../../services/link.service';
import { Title } from '@angular/platform-browser';
import { AuthService } from '../../services/auth.service';
import { PartyService } from '../../services/party.service';

describe('EventDetailComponent', () => {
  let component: EventDetailComponent;
  let fixture: ComponentFixture<EventDetailComponent>;
  let mockApiService: any;
  let mockStarredService: any;
  let mockLinkService: any;
  let mockActivatedRoute: any;
  let mockAuthService: any;
  let mockPartyService: any;

  const mockEvent: Event = {
    eventId: 'BGM101',
    year: 2026,
    active: true,
    title: 'Catan Championship',
    shortDescription: 'Play Catan',
    longDescription: 'Long description',
    categoryCode: 'BGM',
    eventType: 'BGM',
    group: 'Indie',
    orgId: 1,
    gameSystem: { name: 'Catan' },
    rulesEdition: '5th',
    minPlayers: 3,
    maxPlayers: 4,
    ageRequired: '12+',
    experienceRequired: 'None',
    materialsProvided: true,
    startTime: '2026-07-30T10:00:00Z',
    duration: 120,
    endTime: '2026-07-30T12:00:00Z',
    gmNames: 'Alice',
    website: 'http://catan.com',
    email: 'alice@catan.com',
    isTournament: true,
    roundNumber: 1,
    totalRounds: 3,
    minPlayTime: 120,
    attendeeRegistration: 'No',
    cost: 10,
    location: 'ICC',
    roomName: 'Hall A',
    tableNumber: '1',
    ticketsAvailable: 15,
    lastModified: '2026-07-30',
    genconUrl: 'http://gencon.com/BGM101',
    relatedEvents: [
      {
        eventId: 'BGM101',
        ticketsAvailable: 15,
        startTime: '2026-07-30T10:00:00Z', // Thursday
        endTime: '2026-07-30T12:00:00Z'
      },
      {
        eventId: 'BGM102',
        ticketsAvailable: 10,
        startTime: '2026-07-31T14:00:00Z', // Friday
        endTime: '2026-07-31T16:00:00Z'
      },
      {
        eventId: 'BGM103',
        ticketsAvailable: 0,
        startTime: '2026-08-01T10:00:00Z', // Saturday
        endTime: '2026-08-01T12:00:00Z'
      }
    ]
  };

  beforeEach(async () => {
    localStorage.clear();
    mockApiService = {
      getEvent: () => of(mockEvent),
      getPartyTickets: () => of({
        status: 'success',
        tickets: [
          {
            ticketId: 't1',
            eventId: 'BGM999',
            holderEmail: 'friend@example.com',
            ticketStatus: 'active',
            eventStartTime: '2026-07-30T10:00:00Z',
            eventEndTime: '2026-07-30T12:00:00Z'
          }
        ]
      })
    };

    mockStarredService = {
      fetchStarred: () => {},
      isStarred: () => true,
      starredPageData: signal({
        individualEvents: [
          { eventId: 'BGM101', tier: 'must_have', groupTier: 'very_interested', isOverride: false },
          { eventId: 'BGM998', tier: 'purchased', startTime: '2026-07-31T14:00:00Z', endTime: '2026-07-31T16:00:00Z' }
        ]
      }),
      updateTier: () => {},
      removeGroupDefault: () => {},
      removeOverride: () => {}
    };

    mockLinkService = {
      genconUrl: () => 'http://gencon.com',
      getCategoryRouterLink: () => ['/category', 'BGM'],
      getSearchRouterLink: () => ['/search'],
      getEventRouterLink: (eventId: string) => ['/event', eventId]
    };

    mockActivatedRoute = {
      params: of({ eid: 'BGM101' })
    };

    mockAuthService = {
      user: signal({ email: 'leader@example.com', displayName: 'Leader' })
    };

    mockPartyService = {
      parties: signal([
        {
          id: 123,
          name: 'Super Party',
          year: 2026,
          leaderEmail: 'leader@example.com',
          members: [
            { email: 'leader@example.com', displayName: 'Leader' },
            { email: 'friend@example.com', displayName: 'Friend' }
          ]
        }
      ])
    };

    await TestBed.configureTestingModule({
      imports: [EventDetailComponent],
      providers: [
        { provide: ApiService, useValue: mockApiService },
        { provide: StarredService, useValue: mockStarredService },
        { provide: LinkService, useValue: mockLinkService },
        { provide: ActivatedRoute, useValue: mockActivatedRoute },
        { provide: AuthService, useValue: mockAuthService },
        { provide: PartyService, useValue: mockPartyService },
        Title
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(EventDetailComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should load event details on init', () => {
    expect(component.event()).toEqual(mockEvent);
    expect(component.loading()).toBe(false);
  });

  it('should group related events by day of week correctly', () => {
    const groups = component.groupedMatchedEvents();
    expect(groups.length).toBe(3);
    expect(groups[0].day).toBe('Thursday');
    expect(groups[0].events.length).toBe(1);
    expect(groups[1].day).toBe('Friday');
    expect(groups[1].events.length).toBe(1);
    expect(groups[2].day).toBe('Saturday');
    expect(groups[2].events.length).toBe(1);
  });

  it('should check if session is starred and return correct tier', () => {
    expect(component.isSessionStarred('BGM101')).toBe(true);
    expect(component.getEventTier('BGM101')).toBe('must_have');
    expect(component.getGroupTier('BGM101')).toBe('very_interested');
  });

  it('should handle tier click correctly', () => {
    vi.spyOn(mockStarredService, 'updateTier');
    component.handleTierClick('BGM101', 2026, 'somewhat_interested');
    expect(mockStarredService.updateTier).toHaveBeenCalledWith('BGM101', 2026, 'somewhat_interested', false);
  });

  it('should detect time overlaps correctly', () => {
    expect(component.timeOverlaps('2026-07-30T10:00:00Z', '2026-07-30T12:00:00Z', '2026-07-30T11:00:00Z', '2026-07-30T13:00:00Z')).toBe(true);
    expect(component.timeOverlaps('2026-07-30T10:00:00Z', '2026-07-30T12:00:00Z', '2026-07-30T12:00:00Z', '2026-07-30T14:00:00Z')).toBe(false);
  });

  it('should filter out sessions with no tickets when filterHasTickets is active', () => {
    component.filterHasTickets.set(true);
    fixture.detectChanges();
    const matched = component.groupedMatchedEvents();
    const filtered = component.groupedFilteredOutEvents();
    expect(matched.find(g => g.day === 'Saturday')).toBeUndefined();
    expect(matched.find(g => g.day === 'Thursday')).toBeDefined();
    expect(filtered.find(g => g.day === 'Saturday')).toBeDefined();
    expect(filtered.find(g => g.day === 'Thursday')).toBeUndefined();
  });

  it('should filter out overlapping sessions when filterFreeTime is active', () => {
    component.filterFreeTime.set(true);
    fixture.detectChanges();
    const matched = component.groupedMatchedEvents();
    const filtered = component.groupedFilteredOutEvents();
    expect(matched.find(g => g.day === 'Friday')).toBeUndefined();
    expect(filtered.find(g => g.day === 'Friday')).toBeDefined();
  });

  it('should compute party availability counts correctly', () => {
    expect(component.isInParty()).toBe(true);
    expect(component.getOtherAvailableMembersCount(mockEvent.relatedEvents![0])).toBe(0);
    expect(component.getOtherAvailableMembersCount(mockEvent.relatedEvents![1])).toBe(1);
  });

  it('should not filter out sessions when filterPartyAvailable is active', () => {
    component.filterPartyAvailable.set(true);
    fixture.detectChanges();
    const matched = component.groupedMatchedEvents();
    const filtered = component.groupedFilteredOutEvents();
    expect(matched.find(g => g.day === 'Thursday')).toBeDefined();
    expect(matched.find(g => g.day === 'Friday')).toBeDefined();
    expect(filtered.length).toBe(0);
  });

  it('should compute available party member names correctly', () => {
    expect(component.getOtherAvailableMembersNames(mockEvent.relatedEvents![0])).toBe('');
    expect(component.getOtherAvailableMembersNames(mockEvent.relatedEvents![1])).toBe('Friend');
  });

  it('should compute available party members list correctly', () => {
    const listBusy = component.getOtherAvailableMembers(mockEvent.relatedEvents![0]);
    const listFree = component.getOtherAvailableMembers(mockEvent.relatedEvents![1]);
    expect(listBusy.length).toBe(0);
    expect(listFree.length).toBe(1);
    expect(listFree[0].displayName).toBe('Friend');
  });

  it('should read from and write filters to localStorage', () => {
    const setSpy = vi.spyOn(Storage.prototype, 'setItem');

    component.filterHasTickets.set(true);
    component.filterFreeTime.set(true);
    component.filterPartyAvailable.set(true);
    fixture.detectChanges();

    expect(setSpy).toHaveBeenCalledWith('filter_has_tickets', 'true');
    expect(setSpy).toHaveBeenCalledWith('filter_free_time', 'true');
    expect(setSpy).toHaveBeenCalledWith('filter_party_available', 'true');
    setSpy.mockRestore();
  });
});
