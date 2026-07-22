import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA, signal } from '@angular/core';
import { EventCatalogViewComponent } from './event-catalog-view.component';
import { ApiService, EventSummary } from '../../services/api.service';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { AuthService } from '../../services/auth.service';

describe('EventCatalogViewComponent', () => {
  let component: EventCatalogViewComponent;
  let fixture: ComponentFixture<EventCatalogViewComponent>;
  let mockApiService: any;
  let mockAuthService: any;

  const mockEvents: EventSummary[] = [
    {
      anchorEventId: 'EVT1',
      title: 'Acquire',
      shortDescription: 'Classic game',
      categoryCode: 'BGM',
      gameSystem: { name: 'Acquire', bggId: 5, bggRating: 7.5, yearPublished: 1964 },
      wedTickets: 5, wedEvents: 1, wedTotalTickets: 5,
      thuTickets: 5, thuEvents: 1, thuTotalTickets: 5,
      friTickets: 5, friEvents: 1, friTotalTickets: 5,
      satTickets: 5, satEvents: 1, satTotalTickets: 5,
      sunTickets: 5, sunEvents: 1, sunTotalTickets: 5,
      numEvents: 3, orgId: 1
    },
    {
      anchorEventId: 'EVT1-2',
      title: 'Acquire Tournament',
      shortDescription: 'Tournament',
      categoryCode: 'BGM',
      gameSystem: { name: 'Acquire', bggId: 5, bggRating: 7.5, yearPublished: 1964 },
      wedTickets: 10, wedEvents: 1, wedTotalTickets: 10,
      thuTickets: 0, thuEvents: 0, thuTotalTickets: 0,
      friTickets: 0, friEvents: 0, friTotalTickets: 0,
      satTickets: 0, satEvents: 0, satTotalTickets: 0,
      sunTickets: 0, sunEvents: 0, sunTotalTickets: 0,
      numEvents: 1, orgId: 1
    },
    {
      anchorEventId: 'EVT2',
      title: '12 Rivers',
      shortDescription: 'Water game',
      categoryCode: 'BGM',
      gameSystem: { name: '12 Rivers', bggId: 10, bggRating: 6.8, yearPublished: 2019 },
      wedTickets: 0, wedEvents: 1, wedTotalTickets: 0,
      thuTickets: 0, thuEvents: 1, thuTotalTickets: 0,
      friTickets: 0, friEvents: 1, friTotalTickets: 0,
      satTickets: 0, satEvents: 1, satTotalTickets: 0,
      sunTickets: 0, sunEvents: 1, sunTotalTickets: 0,
      numEvents: 2, orgId: 1
    },
    {
      anchorEventId: 'EVT3',
      title: 'Anime Movie',
      shortDescription: 'Fun movie',
      categoryCode: 'ANI',
      gameSystem: { name: 'Anime' },
      wedTickets: 10, wedEvents: 1, wedTotalTickets: 10,
      thuTickets: 10, thuEvents: 1, thuTotalTickets: 10,
      friTickets: 10, friEvents: 1, friTotalTickets: 10,
      satTickets: 10, satEvents: 1, satTotalTickets: 10,
      sunTickets: 10, sunEvents: 1, sunTotalTickets: 10,
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

    await TestBed.configureTestingModule({
      imports: [EventCatalogViewComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        provideRouter([]),
        { provide: ApiService, useValue: mockApiService },
        { provide: AuthService, useValue: mockAuthService },
        {
          provide: ActivatedRoute,
          useValue: {
            params: of({ year: '2026', cat: 'BGM', grouping: 'by_system' }),
            queryParams: of({}),
            snapshot: { params: { year: '2026', cat: 'BGM', grouping: 'by_system' } }
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(EventCatalogViewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }, 30000);

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should display the correct header format (Code: Name) in category mode', () => {
    fixture.componentRef.setInput('mode', 'category');
    component.categoryCode.set('BGM');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const header = compiled.querySelector('h1')?.textContent;
    expect(header).toContain('BGM: Board Games');
  });

  it('should display search header in search mode', () => {
    fixture.componentRef.setInput('mode', 'search');
    component.searchQuery.set('Acquire');
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const header = compiled.querySelector('h1')?.textContent;
    expect(header).toContain('Search Results');
  });

  it('should group events by major category then minor system in search mode', () => {
    fixture.componentRef.setInput('mode', 'search');
    fixture.detectChanges();

    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(2); // ANI and BGM
    const bgmGroup = grouped.find(g => g.name === 'Board Games');
    expect(bgmGroup?.minorGroups.length).toBe(2); // 12 Rivers, Acquire
  });

  it('should sort minor groups numerically (numbers before letters)', () => {
    const grouped = component.groupedEvents();
    const bgmGroup = grouped.find(g => g.name === 'Board Games');
    const minorGroups = bgmGroup!.minorGroups;
    expect(minorGroups[0].minorName).toBe('12 Rivers');
    expect(minorGroups[1].minorName).toBe('Acquire');
  });

  it('should filter sold out events when hideSoldOut is true', () => {
    fixture.componentRef.setInput('mode', 'search');
    component.hideSoldOut.set(true);
    const grouped = component.groupedEvents();
    const bgmGroup = grouped.find(g => g.name === 'Board Games');
    expect(bgmGroup?.minorGroups.length).toBe(1); // 12 Rivers filtered out
    expect(bgmGroup?.minorGroups[0].minorName).toBe('Acquire');
  });

  it('should identify sold out groups correctly', () => {
    const grouped = component.groupedEvents();
    const bgmGroup = grouped.find(g => g.name === 'Board Games');
    const rivers = bgmGroup?.minorGroups.find(g => g.minorName === '12 Rivers');
    const acquire = bgmGroup?.minorGroups.find(g => g.minorName === 'Acquire');
    
    expect(rivers?.isSoldOut).toBe(true);
    expect(acquire?.isSoldOut).toBe(false);
  });

  it('should group events by year descending when groupingMethod is year', () => {
    fixture.componentRef.setInput('mode', 'search');
    component.groupingMethod.set('year');
    const grouped = component.groupedEvents();
    const bgmGroup = grouped.find(g => g.name === 'Board Games');
    expect(bgmGroup?.minorGroups.length).toBe(2);
    expect(bgmGroup?.minorGroups[0].minorName).toBe('2019');
    expect(bgmGroup?.minorGroups[1].minorName).toBe('1964');
  });

  it('should group events by bgg rating descending when groupingMethod is rating', () => {
    fixture.componentRef.setInput('mode', 'search');
    component.groupingMethod.set('rating');
    const grouped = component.groupedEvents();
    const bgmGroup = grouped.find(g => g.name === 'Board Games');
    expect(bgmGroup?.minorGroups.length).toBe(2);
    expect(bgmGroup?.minorGroups[0].minorName).toBe('BGG 7');
    expect(bgmGroup?.minorGroups[1].minorName).toBe('BGG 6');
  });

  it('should filter events by category code in search mode', () => {
    fixture.componentRef.setInput('mode', 'search');
    component.selectedCategories.set(new Set(['ANI']));
    fixture.detectChanges();

    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(1);
    expect(grouped[0].name).toBe('Anime Activities');
  });

  it('should filter events by days and minimum tickets', () => {
    fixture.componentRef.setInput('mode', 'search');
    component.selectedDays.set(new Set(['wed']));
    component.minTickets.set(8);
    fixture.detectChanges();

    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(2);
    const bgmGroup = grouped.find(g => g.name === 'Board Games');
    expect(bgmGroup?.minorGroups.length).toBe(1);
    expect(bgmGroup?.minorGroups[0].minorName).toBe('Acquire');
  });

  it('should filter events by minimum BGG rating', () => {
    fixture.componentRef.setInput('mode', 'search');
    component.minBggRating.set(7.0);
    fixture.detectChanges();

    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(1);
    expect(grouped[0].name).toBe('Board Games');
  });

  it('should filter events by minimum release year', () => {
    component.minYearPublished.set(2000);
    fixture.detectChanges();

    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(1);
    expect(grouped[0].minorGroups[0].minorName).toBe('12 Rivers');
  });

  it('should filter categories auto-complete suggestions correctly', () => {
    component.selectedCategories.set(new Set(['ANI']));
    expect(component.filteredCategories().length).toBe(20);

    component.categorySearchQuery.set('board');
    const filtered = component.filteredCategories();
    expect(filtered.length).toBe(1);
    expect(filtered[0].code).toBe('BGM');
  });

  it('should correctly parse year from route params without NaN', () => {
    expect(component.year()).toBe(2026);
    expect(isNaN(component.year())).toBe(false);
  });
});
