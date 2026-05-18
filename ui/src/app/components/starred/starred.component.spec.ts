import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA, signal } from '@angular/core';
import { StarredComponent } from './starred.component';
import { StarredService } from '../../services/starred.service';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { LinkService } from '../../services/link.service';
import { ActivatedRoute, Router } from '@angular/router';
import { of } from 'rxjs';
import { describe, it, expect, beforeEach, vi } from 'vitest';

describe('StarredComponent', () => {
  let component: StarredComponent;
  let fixture: ComponentFixture<StarredComponent>;
  let mockStarredService: any;

  beforeEach(async () => {
    mockStarredService = {
      starredPageData: signal({
        email: 'test@example.com',
        year: 2026,
        calendarEvents: [],
        individualEvents: [
          {
            eventId: 'BGM26ND100001',
            title: 'Catan',
            shortDescription: 'Trading game',
            categoryCode: 'BGM',
            startTime: '2026-07-30T10:00:00Z',
            endTime: '2026-07-30T12:00:00Z',
            tier: 'very_interested',
            groupTier: 'very_interested',
            isOverride: false
          }
        ],
        metadata: { startDate: '2026-07-30', endDate: '2026-08-02' },
        starredClusters: [],
        starredEvents: []
      }),
      fetchStarred: vi.fn(),
      updateTier: vi.fn(),
      removeOverride: vi.fn()
    };

    const mockApiService = {
      getWishlistConstraints: vi.fn().mockReturnValue(of([])),
      getWishlist: vi.fn().mockReturnValue(of([]))
    };

    const mockAuthService = {
      user: vi.fn().mockReturnValue({ email: 'test@example.com' }),
      authLoaded: vi.fn().mockReturnValue(true)
    };

    const mockActivatedRoute = {
      params: of({ year: '2026', tab: 'list' })
    };

    const mockRouter = {
      navigate: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [StarredComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        { provide: StarredService, useValue: mockStarredService },
        { provide: ApiService, useValue: mockApiService },
        { provide: AuthService, useValue: mockAuthService },
        { provide: ActivatedRoute, useValue: mockActivatedRoute },
        { provide: Router, useValue: mockRouter },
        LinkService
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(StarredComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }, 30000);

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should call starredService.updateTier with starGroup=true when updateGroupTier is called', () => {
    component.updateGroupTier('BGM26ND100001', 'must_have');
    expect(mockStarredService.updateTier).toHaveBeenCalledWith('BGM26ND100001', 2026, 'must_have', true);
  });

  it('should call starredService.removeOverride when resetOverride is called', () => {
    component.resetOverride('BGM26ND100001');
    expect(mockStarredService.removeOverride).toHaveBeenCalledWith('BGM26ND100001', 2026);
  });
});
