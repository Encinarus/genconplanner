import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA, signal } from '@angular/core';
import { SearchComponent } from './search.component';
import { ApiService } from '../../services/api.service';
import { ActivatedRoute, provideRouter, Router } from '@angular/router';
import { of } from 'rxjs';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { EventSummary } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { StarredService } from '../../services/starred.service';

describe('SearchComponent', () => {
  let component: SearchComponent;
  let fixture: ComponentFixture<SearchComponent>;
  let mockApiService: any;
  let mockAuthService: any;
  let mockStarredService: any;
  let router: Router;

  const mockEvents: EventSummary[] = [
    {
      anchorEventId: 'EVT1',
      title: 'Acquire',
      shortDescription: 'Classic game',
      categoryCode: 'BGM',
      gameSystem: { name: 'Acquire', bggId: 5, bggRating: 7.5, yearPublished: 1964 },
      wedTickets: 5, thuTickets: 5, friTickets: 5, satTickets: 5, sunTickets: 5,
      numEvents: 1, orgId: 1
    },
    {
      anchorEventId: 'EVT3',
      title: 'Anime Movie',
      shortDescription: 'Fun movie',
      categoryCode: 'ANI',
      gameSystem: { name: 'Anime' },
      wedTickets: 10, thuTickets: 10, friTickets: 10, satTickets: 10, sunTickets: 10,
      numEvents: 1, orgId: 1
    }
  ];

  beforeEach(async () => {
    mockApiService = {
      searchEvents: vi.fn().mockReturnValue(of(mockEvents))
    };

    mockAuthService = {
      user: signal({ email: 'test@example.com' })
    };

    mockStarredService = {
      fetchStarred: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [SearchComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        provideRouter([]),
        { provide: ApiService, useValue: mockApiService },
        { provide: AuthService, useValue: mockAuthService },
        { provide: StarredService, useValue: mockStarredService },
        {
          provide: ActivatedRoute,
          useValue: {
            queryParams: of({ q: 'test', year: '2026' }),
            params: of({ grouping: 'by_system' }),
            snapshot: { 
              queryParams: { q: 'test', year: '2026' },
              params: { grouping: 'by_system' }
            }
          }
        }
      ]
    })
    .compileComponents();

    router = TestBed.inject(Router);
    fixture = TestBed.createComponent(SearchComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }, 30000);

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should group events by major category then minor system', () => {
    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(2); // ANI and BGM
    expect(grouped.find(g => g.name === 'Anime Activities')?.minorGroups.length).toBe(1);
    expect(grouped.find(g => g.name === 'Board Games')?.minorGroups.length).toBe(1);
  });

  it('should filter sold out events when hideSoldOut is true', () => {
    const eventsWithSoldOut = [
      ...mockEvents,
      {
        anchorEventId: 'EVT2',
        title: 'Sold Out Game',
        shortDescription: 'No tickets',
        categoryCode: 'BGM',
        gameSystem: { name: 'Dead Game' },
        wedTickets: 0, thuTickets: 0, friTickets: 0, satTickets: 0, sunTickets: 0,
        numEvents: 1, orgId: 1
      }
    ];
    component.events.set(eventsWithSoldOut);
    
    expect(component.filteredGroupsCount()).toBe(3);
    
    component.hideSoldOut.set(true);
    expect(component.filteredGroupsCount()).toBe(2);
  });

  it('should group events by year descending when groupingMethod is year', () => {
    component.groupingMethod.set('year');
    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(2);
    const bgmGroup = grouped.find(g => g.name === 'Board Games');
    expect(bgmGroup?.minorGroups.length).toBe(1);
    expect(bgmGroup?.minorGroups[0].minorName).toBe('1964');
  });

  it('should group events by bgg rating descending when groupingMethod is rating', () => {
    component.groupingMethod.set('rating');
    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(2);
    const bgmGroup = grouped.find(g => g.name === 'Board Games');
    expect(bgmGroup?.minorGroups.length).toBe(1);
    expect(bgmGroup?.minorGroups[0].minorName).toBe('BGG 7');
  });

  it('should filter events by category code', () => {
    component.selectedCategories.set(new Set(['ANI']));
    fixture.detectChanges();
    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(1);
    expect(grouped[0].name).toBe('Anime Activities');
  });

  it('should filter events by days and minimum tickets', () => {
    // Acquire has 5 tickets per day, Anime Movie has 10 tickets per day.
    // Filter with minTickets = 8 on Wed: Acquire should be excluded, Anime Movie should remain.
    component.selectedDays.set(new Set(['wed']));
    component.minTickets.set(8);
    fixture.detectChanges();

    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(1);
    expect(grouped[0].name).toBe('Anime Activities');
  });

  it('should filter events by minimum BGG rating', () => {
    // Acquire has BGG rating 7.5, Anime Movie has no rating.
    component.minBggRating.set(7.0);
    fixture.detectChanges();

    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(1);
    expect(grouped[0].name).toBe('Board Games');
  });

  it('should filter events by minimum release year', () => {
    // Acquire has release year 1964, Anime Movie has no release year.
    component.minYearPublished.set(1980);
    fixture.detectChanges();

    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(0);
  });

  it('should filter categories auto-complete suggestions correctly', () => {
    // categorySearchQuery is empty, should return all categories except selected
    component.selectedCategories.set(new Set(['ANI']));
    expect(component.filteredCategories().length).toBe(20); // 21 total - 1 selected

    // type 'board'
    component.categorySearchQuery.set('board');
    const filtered = component.filteredCategories();
    expect(filtered.length).toBe(1);
    expect(filtered[0].code).toBe('BGM');

    // selectCategory
    component.selectCategory('BGM');
    expect(component.selectedCategories().has('BGM')).toBe(true);
    expect(component.categorySearchQuery()).toBe('');
  });

  it('should navigate with correct query params when toggleDay/toggleCategory/toggleFilterFree is called', () => {
    // Reset signals to default state
    component.selectedCategories.set(new Set());
    component.selectedDays.set(new Set(['wed', 'thu', 'fri', 'sat', 'sun']));
    component.filterFree.set(false);

    const navSpy = vi.spyOn(router, 'navigate');

    component.toggleFilterFree();
    expect(navSpy).toHaveBeenCalledWith([], expect.objectContaining({
      queryParams: expect.objectContaining({ free: 'true' })
    }));

    component.toggleDay('wed'); // Wednesday was in the set, now it gets removed
    expect(navSpy).toHaveBeenCalledWith([], expect.objectContaining({
      queryParams: expect.objectContaining({ days: 'thu,fri,sat,sun' })
    }));

    // Reset categories again to be empty before testing category toggling
    component.selectedCategories.set(new Set());
    component.toggleCategory('BGM');
    expect(navSpy).toHaveBeenCalledWith([], expect.objectContaining({
      queryParams: expect.objectContaining({ cats: 'BGM' })
    }));
  });
});
