import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, of, tap } from 'rxjs';

export interface Category {
  name: string;
  code: string;
  eventCount: number;
  year: number;
}

export interface EventSummary {
  anchorEventId: string;
  title: string;
  shortDescription: string;
  numEvents: number;
  wedTickets: number;
  thuTickets: number;
  friTickets: number;
  satTickets: number;
  sunTickets: number;
  orgId: number;
  categoryCode: string;
  gameSystem: {
    name: string;
    bggId?: number;
    bggRating?: number;
    numBggRatings?: number;
    yearPublished?: number;
  };
}

export interface SearchParams {
  year: number;
  cat?: string;
  search?: string;
  org_id?: number;
}

export interface CalendarEvent {
  title: string;
  startTime: string;
  endTime: string;
  genconUrl: string;
  plannerUrl: string;
  shortCategory: string;
  shortDescription: string;
  similarCount: number;
}

export interface StarredEventDetail {
  eventId: string;
  title: string;
  shortDescription: string;
  categoryCode: string;
  startTime: string;
  endTime: string;
  genconUrl: string;
  plannerUrl: string;
  tier: string;
  groupTier: string;
  isOverride: boolean;
}

export interface CalendarMetadata {
  startDate: string;
  endDate: string;
}

export interface WishlistItem {
  event: StarredEventDetail;
  status: string;
  reasoning: string[];
  score: number;
}

export interface WishlistConstraint {
  dayOfWeek: number; // -1 for Every Day, 0-6 for Sun-Sat
  startHour: number;
  startMinute: number;
  endHour: number;
  endMinute: number;
  minDurationMinutes: number;
}

export interface StarredPageData {
  email: string;
  year: number;
  calendarEvents: CalendarEvent[];
  individualEvents: StarredEventDetail[];
  metadata: CalendarMetadata;
  starredClusters: string[];
  starredEvents: string[];
}

export interface PartyMember {
  displayName: string;
  email: string;
}

export interface Party {
  id: number;
  name: string;
  year: number;
  leaderEmail: string;
  shortCode: string;
  inviteLink: string;
  members: PartyMember[];
}

export interface MemberInterest {
  email: string;
  displayName: string;
  tier: string;
}

export interface SharedInterestGroup {
  clusterId: string;
  repEventId: string;
  title: string;
  shortCategory: string;
  gameSystem: string;
  totalSessions: number;
  totalTickets: number;
  memberInterests: MemberInterest[];
  groupScore: number;
}

export interface Event {
  eventId: string;
  year: number;
  active: boolean;
  title: string;
  shortDescription: string;
  longDescription: string;
  categoryCode: string;
  eventType: string;
  group: string;
  orgId: number;
  gameSystem: {
    name: string;
    bggId?: number;
    bggRating?: number;
    numBggRatings?: number;
    yearPublished?: number;
  };
  rulesEdition: string;
  minPlayers: number;
  maxPlayers: number;
  ageRequired: string;
  experienceRequired: string;
  materialsProvided: boolean;
  startTime: string;
  duration: number;
  endTime: string;
  gmNames: string;
  website: string;
  email: string;
  isTournament: boolean;
  roundNumber: number;
  totalRounds: number;
  minPlayTime: number;
  attendeeRegistration: string;
  cost: number;
  location: string;
  roomName: string;
  tableNumber: string;
  ticketsAvailable: number;
  lastModified: string;
  genconUrl: string;
  relatedEvents?: {
    eventId: string;
    ticketsAvailable: number;
    startTime: string;
    endTime: string;
  }[];
}

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private categoriesCache = new Map<number, Category[]>();
  private eventSummariesCache = new Map<string, EventSummary[]>();
  private eventDetailsCache = new Map<string, Event>();

  constructor(private http: HttpClient) { }

  getCategories(year: number): Observable<Category[]> {
    if (this.categoriesCache.has(year)) {
      return of(this.categoriesCache.get(year)!);
    }
    return this.http.get<Category[]>(`/api/v1/category/${year}`).pipe(
      tap(data => this.categoriesCache.set(year, data))
    );
  }

  searchEvents(params: SearchParams): Observable<EventSummary[]> {
    const cacheKey = JSON.stringify(params);
    if (this.eventSummariesCache.has(cacheKey)) {
      return of(this.eventSummariesCache.get(cacheKey)!);
    }

    let httpParams = new HttpParams().set('year', params.year.toString());
    if (params.search) httpParams = httpParams.set('search', params.search);
    if (params.cat) httpParams = httpParams.set('cat', params.cat);
    if (params.org_id) httpParams = httpParams.set('org_id', params.org_id.toString());
    
    return this.http.get<EventSummary[]>(`/api/v1/events`, { params: httpParams }).pipe(
      tap(data => this.eventSummariesCache.set(cacheKey, data))
    );
  }

  getEvent(eventId: string): Observable<Event> {
    if (this.eventDetailsCache.has(eventId)) {
      return of(this.eventDetailsCache.get(eventId)!);
    }
    return this.http.get<Event>(`/api/v1/event/${eventId}`).pipe(
      tap(data => this.eventDetailsCache.set(eventId, data))
    );
  }

  getUserEvents(email: string, year: number): Observable<any> {
    return this.http.get<any>(`/api/v1/user/events/${email}/${year}`);
  }

  getStarredEvents(year: number): Observable<EventSummary[]> {
    return this.http.get<EventSummary[]>(`/api/v1/user/starred/${year}`);
  }

  starEvent(eventId: string, add: boolean, related: boolean, tier: string = ''): Observable<any> {
    return this.http.post<any>(`/api/v1/user/star`, { eventId, add, related, tier });
  }

  getAgenda(year: number): Observable<StarredEventDetail[]> {
    return this.http.get<StarredEventDetail[]>(`/api/v1/user/agenda/${year}`);
  }

  getStarredCalendarEvents(year: number): Observable<CalendarEvent[]> {
    return this.http.get<CalendarEvent[]>(`/api/v1/user/starred/calendar/${year}`);
  }

  getStarredIndividualEvents(year: number): Observable<StarredEventDetail[]> {
    return this.http.get<StarredEventDetail[]>(`/api/v1/user/starred/list/${year}`);
  }

  getCalendarMetadata(year: number): Observable<CalendarMetadata> {
    return this.http.get<CalendarMetadata>(`/api/v1/calendar/metadata/${year}`);
  }

  getStarredPageData(year: number): Observable<StarredPageData> {
    return this.http.get<StarredPageData>(`/api/v1/user/starred/page/${year}`);
  }

  bulkClearStarred(year: number): Observable<any> {
    return this.http.post<any>(`/api/v1/user/starred/clear/${year}`, {});
  }

  bulkReplaceStarred(year: number, text: string, overwrite: boolean, asGroups: boolean): Observable<any> {
    return this.http.post<any>(`/api/v1/user/starred/bulk/${year}`, { text, overwrite, asGroups });
  }

  getParties(): Observable<Party[]> {
    return this.http.get<Party[]>(`/api/v1/user/parties`);
  }

  createParty(name: string, year: number): Observable<Party> {
    return this.http.post<Party>(`/api/v1/user/parties`, { name, year });
  }

  getParty(id: number | string): Observable<Party> {
    return this.http.get<Party>(`/api/v1/party/${id}`);
  }

  getPartyInterests(id: number | string, year: number): Observable<SharedInterestGroup[]> {
    return this.http.get<SharedInterestGroup[]>(`/api/v1/party/${id}/interests`, { params: new HttpParams().set('year', year.toString()) });
  }

  renameParty(id: number, name: string): Observable<any> {
    return this.http.post<any>(`/api/v1/party/${id}/rename`, { name });
  }

  transferLeadership(id: number, newLeaderEmail: string): Observable<any> {
    return this.http.post<any>(`/api/v1/party/${id}/transfer`, { newLeaderEmail });
  }

  joinParty(id: string): Observable<any> {
    return this.http.post<any>(`/api/v1/party/${id}/join`, {});
  }

  leaveParty(id: number): Observable<any> {
    return this.http.post<any>(`/api/v1/party/${id}/leave`, {});
  }

  deleteParty(id: number): Observable<any> {
    return this.http.delete<any>(`/api/v1/party/${id}`);
  }

  renameUser(displayName: string): Observable<any> {
    return this.http.post<any>(`/api/v1/user/rename`, { displayName });
  }

  getLastUpdate(): Observable<{lastUpdate: string}> {
    return this.http.get<{lastUpdate: string}>(`/api/v1/metadata/last_update`);
  }

  getWishlist(year: number): Observable<WishlistItem[]> {
    return this.http.get<WishlistItem[]>(`/api/v1/user/wishlist/${year}`);
  }

  getWishlistConstraints(): Observable<WishlistConstraint[]> {
    return this.http.get<WishlistConstraint[]>(`/api/v1/user/wishlist/constraints`);
  }

  updateWishlistConstraints(constraints: WishlistConstraint[]): Observable<any> {
    return this.http.post<any>(`/api/v1/user/wishlist/constraints`, constraints);
  }
}
