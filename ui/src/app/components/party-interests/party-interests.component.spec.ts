import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA, signal } from '@angular/core';
import { PartyInterestsComponent } from './party-interests.component';
import { ApiService, SharedInterestGroup, Event } from '../../services/api.service';
import { PartyStreamService } from '../../services/party-stream.service';
import { AuthService } from '../../services/auth.service';
import { LinkService } from '../../services/link.service';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { describe, it, expect, beforeEach, vi } from 'vitest';

describe('PartyInterestsComponent', () => {
  let component: PartyInterestsComponent;
  let fixture: ComponentFixture<PartyInterestsComponent>;
  let mockApiService: any;
  let mockPartyStreamService: any;
  let mockAuthService: any;
  let mockLinkService: any;

  const mockGroups: SharedInterestGroup[] = [
    {
      clusterId: 'c1',
      repEventId: 'e1',
      title: 'Catan Championship',
      shortCategory: 'BGM',
      gameSystem: 'Catan',
      totalSessions: 4,
      totalTickets: 20,
      memberInterests: [
        { email: 'test@example.com', displayName: 'Test User', tier: 'must_have' },
        { email: 'other@example.com', displayName: 'Other User', tier: 'very_interested' }
      ],
      groupScore: 150
    },
    {
      clusterId: 'c2',
      repEventId: 'e2',
      title: 'D&D Adventurers League',
      shortCategory: 'RPG',
      gameSystem: 'Dungeons & Dragons',
      totalSessions: 10,
      totalTickets: 60,
      memberInterests: [
        { email: 'other@example.com', displayName: 'Other User', tier: 'must_have' }
      ],
      groupScore: 100
    },
    {
      clusterId: 'c3',
      repEventId: 'e3',
      title: 'General Board Gaming',
      shortCategory: 'BGM',
      gameSystem: '', // empty system should group under General / Unspecified
      totalSessions: 2,
      totalTickets: 10,
      memberInterests: [],
      groupScore: 0
    }
  ];

  const mockEventDetails: Event = {
    eventId: 'e1',
    year: 2026,
    active: true,
    title: 'Catan Championship',
    shortDescription: 'Compete for the Catan crown',
    longDescription: 'Full tournament rules apply.',
    categoryCode: 'BGM',
    eventType: 'Board Game',
    group: 'Catan Studio',
    orgId: 101,
    gameSystem: {
      name: 'Catan',
      bggId: 13,
      bggRating: 7.1,
      numBggRatings: 120000,
      yearPublished: 1995
    },
    rulesEdition: '5th',
    minPlayers: 3,
    maxPlayers: 4,
    ageRequired: '12+',
    experienceRequired: 'Some',
    materialsProvided: true,
    startTime: '2026-08-01T10:00:00Z',
    duration: 4,
    endTime: '2026-08-01T14:00:00Z',
    gmNames: 'John Doe',
    website: 'catanstudio.com',
    email: 'info@catanstudio.com',
    isTournament: true,
    roundNumber: 1,
    totalRounds: 3,
    minPlayTime: 120,
    attendeeRegistration: 'Yes',
    cost: 10,
    location: 'ICC',
    roomName: 'Hall A',
    tableNumber: '1-5',
    ticketsAvailable: 20,
    lastModified: '2026-05-01T00:00:00Z',
    genconUrl: 'https://gencon.com/events/e1'
  };

  beforeEach(async () => {
    mockApiService = {
      getPartyInterests: vi.fn().mockReturnValue(of(mockGroups)),
      getEvent: vi.fn().mockReturnValue(of(mockEventDetails)),
      starEvent: vi.fn().mockReturnValue(of({ success: true }))
    };

    mockPartyStreamService = {
      connect: vi.fn(),
      disconnect: vi.fn(),
      latestInterestUpdate: signal<any>(null),
      streamResumed: signal<number>(0)
    };

    mockAuthService = {
      user: signal<{ email: string; displayName: string } | null>({
        email: 'test@example.com',
        displayName: 'Test User'
      })
    };

    mockLinkService = {
      getEventRouterLink: vi.fn().mockReturnValue(['/events', 'e1']),
      getSearchRouterLink: vi.fn().mockReturnValue(['/search'])
    };

    await TestBed.configureTestingModule({
      imports: [PartyInterestsComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        provideRouter([]),
        { provide: ApiService, useValue: mockApiService },
        { provide: PartyStreamService, useValue: mockPartyStreamService },
        { provide: AuthService, useValue: mockAuthService },
        { provide: LinkService, useValue: mockLinkService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PartyInterestsComponent);
    component = fixture.componentInstance;
    component.partyId = 1;
    component.year = 2026;
    fixture.detectChanges();
  }, 30000);

  it('should create and load interests on init', () => {
    expect(component).toBeTruthy();
    expect(mockApiService.getPartyInterests).toHaveBeenCalledWith(1, 2026);
    expect(component.groups().length).toBe(3);
    expect(component.selectedGroup()?.clusterId).toBe('c1');
    expect(component.selectedEventDetails()?.eventId).toBe('e1');
  });

  it('should correctly group filtered events by event category alphabetically', () => {
    const catGroups = component.filteredGroupsByCategory();
    expect(catGroups.length).toBe(2);

    // Alphabetical order: Board Games (c1, c3), Role Playing Games (c2)
    expect(catGroups[0].categoryName).toBe('Board Games');
    expect(catGroups[0].groups.length).toBe(2);

    expect(catGroups[1].categoryName).toBe('Role Playing Games');
    expect(catGroups[1].groups.length).toBe(1);
  });

  it('should calculate the total filtered count correctly', () => {
    expect(component.getTotalFilteredCount()).toBe(3);
  });

  it('should filter events by myInterestFilter correctly', () => {
    // Filter to must_have (Test User has must_have on c1)
    component.myInterestFilter.set('must_have');
    let catGroups = component.filteredGroupsByCategory();
    expect(component.getTotalFilteredCount()).toBe(1);
    expect(catGroups[0].groups[0].clusterId).toBe('c1');

    // Filter to unrated (Test User has no rating on c2 and c3)
    component.myInterestFilter.set('unrated');
    catGroups = component.filteredGroupsByCategory();
    expect(component.getTotalFilteredCount()).toBe(2);
    expect(catGroups.some(s => s.groups.some(g => g.clusterId === 'c2'))).toBe(true);
    expect(catGroups.some(s => s.groups.some(g => g.clusterId === 'c3'))).toBe(true);
  });

  it('should correctly identify user tier and other members', () => {
    const group = component.groups()[0]; // c1
    expect(component.getUserTier(group)).toBe('must_have');

    const otherMembers = component.getOtherMembers(group);
    expect(otherMembers.length).toBe(1);
    expect(otherMembers[0].email).toBe('other@example.com');
    expect(component.getOtherMemberCountByTier(group, 'very_interested')).toBe(1);
  });

  it('should handle optimistic starring correctly', () => {
    const group = component.groups()[1]; // c2 (Test User is currently unrated)
    expect(component.getUserTier(group)).toBe('');

    component.onStar(group, 'somewhat_interested');
    expect(mockApiService.starEvent).toHaveBeenCalledWith('e2', true, true, 'somewhat_interested');

    const updatedGroup = component.groups().find(g => g.clusterId === 'c2')!;
    expect(component.getUserTier(updatedGroup)).toBe('somewhat_interested');
  });

  it('should handle real-time SSE updates correctly', () => {
    // Simulate real-time update from another user on c3
    mockPartyStreamService.latestInterestUpdate.set({
      party_id: 1,
      cluster_id: 'c3',
      email: 'newuser@example.com',
      tier: 'very_interested'
    });
    fixture.detectChanges();

    const updatedGroup = component.groups().find(g => g.clusterId === 'c3')!;
    expect(updatedGroup.memberInterests.length).toBe(1);
    expect(updatedGroup.memberInterests[0].email).toBe('newuser@example.com');
    expect(updatedGroup.memberInterests[0].tier).toBe('very_interested');
    expect(updatedGroup.groupScore).toBe(50); // very_interested adds 50 points
  });
});
