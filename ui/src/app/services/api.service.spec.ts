import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { ApiService, Category, EventSummary, Event } from './api.service';

describe('ApiService', () => {
  let service: ApiService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [ApiService]
    });
    service = TestBed.inject(ApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should fetch categories and cache them', () => {
    const mockCategories: Category[] = [
      { name: 'Board Games', code: 'BGM', eventCount: 100, year: 2026 }
    ];

    // First call should hit HTTP
    service.getCategories(2026).subscribe(categories => {
      expect(categories).toEqual(mockCategories);
    });

    const req = httpMock.expectOne('/api/v1/category/2026');
    expect(req.request.method).toBe('GET');
    req.flush(mockCategories);

    // Second call should hit Cache (no HTTP request)
    service.getCategories(2026).subscribe(categories => {
      expect(categories).toEqual(mockCategories);
    });

    httpMock.expectNone('/api/v1/category/2026');
  });

  it('should search events and cache results', () => {
    const mockSummaries: EventSummary[] = [
      {
        anchorEventId: 'BGM101',
        title: 'Catan',
        shortDescription: 'Play Catan',
        numEvents: 2,
        wedTickets: 0,
        thuTickets: 10,
        friTickets: 10,
        satTickets: 0,
        sunTickets: 0,
        orgId: 1,
        categoryCode: 'BGM',
        gameSystem: { name: 'Catan' }
      }
    ];

    const params = { year: 2026, cat: 'BGM' };

    service.searchEvents(params).subscribe(summaries => {
      expect(summaries).toEqual(mockSummaries);
    });

    const req = httpMock.expectOne(req => req.url.includes('/api/v1/events') && req.params.get('cat') === 'BGM');
    expect(req.request.method).toBe('GET');
    req.flush(mockSummaries);

    // Second call should hit Cache
    service.searchEvents(params).subscribe(summaries => {
      expect(summaries).toEqual(mockSummaries);
    });

    httpMock.expectNone(req => req.url.includes('/api/v1/events'));
  });

  it('should fetch single event details and cache them', () => {
    const mockEvent = {
      eventId: 'BGM101',
      year: 2026,
      active: true,
      title: 'Catan',
      shortDescription: 'Play Catan',
      longDescription: 'Long desc',
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
      genconUrl: 'http://gencon.com/BGM101'
    };

    service.getEvent('BGM101').subscribe(event => {
      expect(event).toEqual(mockEvent);
    });

    const req = httpMock.expectOne('/api/v1/event/BGM101');
    expect(req.request.method).toBe('GET');
    req.flush(mockEvent);

    // Second call hits cache
    service.getEvent('BGM101').subscribe(event => {
      expect(event).toEqual(mockEvent);
    });

    httpMock.expectNone('/api/v1/event/BGM101');
  });

  it('should call starEvent endpoint correctly', () => {
    service.starEvent('BGM101', true, false, 'must_have', false).subscribe(res => {
      expect(res.success).toBe(true);
    });

    const req = httpMock.expectOne('/api/v1/user/star');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({
      eventId: 'BGM101',
      add: true,
      related: false,
      tier: 'must_have',
      removeAll: false
    });
    req.flush({ success: true });
  });
});
