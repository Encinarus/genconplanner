import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA } from '@angular/core';
import { SearchComponent } from './search.component';
import { ApiService } from '../../services/api.service';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { EventSummary } from '../../services/api.service';

describe('SearchComponent', () => {
  let component: SearchComponent;
  let fixture: ComponentFixture<SearchComponent>;
  let mockApiService: any;

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

    await TestBed.configureTestingModule({
      imports: [SearchComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        provideRouter([]),
        { provide: ApiService, useValue: mockApiService },
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
    // Add a sold out event to mock data
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
    mockApiService.searchEvents.mockReturnValue(of(eventsWithSoldOut));
    
    // Trigger re-load by re-initializing or manually setting the signal
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
});
