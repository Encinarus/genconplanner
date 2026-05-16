import { TestBed } from '@angular/core/testing';
import { StarredService } from './starred.service';
import { ApiService } from './api.service';
import { AuthService } from './auth.service';
import { of } from 'rxjs';
import { describe, it, expect, beforeEach, vi } from 'vitest';

describe('StarredService', () => {
  let service: StarredService;
  let mockApiService: any;
  let mockAuthService: any;

  beforeEach(() => {
    mockApiService = {
      starEvent: vi.fn().mockReturnValue(of({})),
      getStarredPageData: vi.fn().mockReturnValue(of({
        email: 'test@example.com',
        year: 2026,
        calendarEvents: [],
        individualEvents: [],
        metadata: { startDate: '', endDate: '' },
        starredClusters: ['BGM1'],
        starredEvents: ['BGM2']
      }))
    };

    mockAuthService = {
      user: vi.fn().mockReturnValue({ email: 'test@example.com', displayName: 'Test' }),
      signIn: vi.fn()
    };

    TestBed.configureTestingModule({
      providers: [
        StarredService,
        { provide: ApiService, useValue: mockApiService },
        { provide: AuthService, useValue: mockAuthService }
      ]
    });

    service = TestBed.inject(StarredService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should update signals correctly from StarredPageData', () => {
    service.fetchStarred(2026, true);
    expect(service.groupStarredIds()).toEqual(['BGM1']);
    expect(service.starredIds()).toEqual(['BGM1', 'BGM2']);
  });

  it('should call api.starEvent with correct parameters when updateTier is called', () => {
    service.updateTier('BGM1', 2026, 'must_have', true);
    expect(mockApiService.starEvent).toHaveBeenCalledWith('BGM1', true, true, 'must_have');
  });

  it('should call api.starEvent with correct parameters when removeOverride is called', () => {
    service.removeOverride('BGM2', 2026);
    expect(mockApiService.starEvent).toHaveBeenCalledWith('BGM2', false, false, '');
  });
});
