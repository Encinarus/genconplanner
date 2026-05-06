import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA } from '@angular/core';
import { CategoryDetailComponent } from './category-detail.component';
import { ApiService } from '../../services/api.service';
import { ActivatedRoute, Router, provideRouter, RouterModule } from '@angular/router';
import { of } from 'rxjs';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { EventSummary } from '../../services/api.service';

describe('CategoryDetailComponent', () => {
  let component: CategoryDetailComponent;
  let fixture: ComponentFixture<CategoryDetailComponent>;
  let mockApiService: any;

  const mockEvents: EventSummary[] = [
    {
      anchorEventId: 'EVT1',
      title: 'Acquire',
      shortDescription: 'Classic game',
      categoryCode: 'BGM',
      gameSystem: { name: 'Acquire', bggId: 5, bggRating: 7.5, yearPublished: 1964 },
      wedTickets: 5, thuTickets: 5, friTickets: 5, satTickets: 5, sunTickets: 5,
      numEvents: 3, orgId: 1
    },
    {
      anchorEventId: 'EVT1-2',
      title: 'Acquire Tournament',
      shortDescription: 'Tournament',
      categoryCode: 'BGM',
      gameSystem: { name: 'Acquire', bggId: 5, bggRating: 7.5, yearPublished: 1964 },
      wedTickets: 10, thuTickets: 0, friTickets: 0, satTickets: 0, sunTickets: 0,
      numEvents: 1, orgId: 1
    },
    {
      anchorEventId: 'EVT2',
      title: '12 Rivers',
      shortDescription: 'Water game',
      categoryCode: 'BGM',
      gameSystem: { name: '12 Rivers', bggId: 10, bggRating: 6.8, yearPublished: 2019 },
      wedTickets: 0, thuTickets: 0, friTickets: 0, satTickets: 0, sunTickets: 0,
      numEvents: 2, orgId: 1
    }
  ];

  beforeEach(async () => {
    mockApiService = {
      searchEvents: vi.fn().mockReturnValue(of(mockEvents))
    };

    await TestBed.configureTestingModule({
      imports: [CategoryDetailComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        provideRouter([]),
        { provide: ApiService, useValue: mockApiService },
        {
          provide: ActivatedRoute,
          useValue: {
            params: of({ year: '2026', cat: 'BGM' }),
            snapshot: { params: { year: '2026', cat: 'BGM' } }
          }
        }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CategoryDetailComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should display the correct header format (Code: Name)', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const header = compiled.querySelector('h1')?.textContent;
    expect(header).toContain('BGM: Board Games');
  });

  it('should group events by game system', () => {
    const grouped = component.groupedEvents();
    expect(grouped.length).toBe(1); // One major group (Board Games)
    expect(grouped[0].minorGroups.length).toBe(2); // Two minor groups (12 Rivers, Acquire)
  });

  it('should sort groups numerically (numbers before letters)', () => {
    const grouped = component.groupedEvents();
    const minorGroups = grouped[0].minorGroups;
    expect(minorGroups[0].minorName).toBe('12 Rivers');
    expect(minorGroups[1].minorName).toBe('Acquire');
  });

  it('should filter sold out events when hideSoldOut is true', () => {
    component.hideSoldOut.set(true);
    const grouped = component.groupedEvents();
    expect(grouped[0].minorGroups.length).toBe(1);
    expect(grouped[0].minorGroups[0].minorName).toBe('Acquire');
    expect(component.filteredGroupsCount()).toBe(2); // Acquire + Acquire Tournament
    expect(component.totalSessionsCount()).toBe(4); // 3 + 1
  });

  it('should identify sold out groups correctly', () => {
    const grouped = component.groupedEvents();
    const rivers = grouped[0].minorGroups.find(g => g.minorName === '12 Rivers');
    const acquire = grouped[0].minorGroups.find(g => g.minorName === 'Acquire');
    
    expect(rivers?.isSoldOut).toBe(true);
    expect(acquire?.isSoldOut).toBe(false);
  });
});
