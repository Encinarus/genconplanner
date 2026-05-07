import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';

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
}

export interface CalendarMetadata {
  startDate: string;
  endDate: string;
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
  constructor(private http: HttpClient) { }

  getCategories(year: number): Observable<Category[]> {
    return this.http.get<Category[]>(`/api/v1/category/${year}`);
  }

  searchEvents(params: SearchParams): Observable<EventSummary[]> {
    let httpParams = new HttpParams().set('year', params.year.toString());
    if (params.search) httpParams = httpParams.set('search', params.search);
    if (params.cat) httpParams = httpParams.set('cat', params.cat);
    if (params.org_id) httpParams = httpParams.set('org_id', params.org_id.toString());
    return this.http.get<EventSummary[]>(`/api/v1/events`, { params: httpParams });
  }

  getEvent(eventId: string): Observable<Event> {
    return this.http.get<Event>(`/api/v1/event/${eventId}`);
  }

  getUserEvents(email: string, year: number): Observable<any> {
    return this.http.get<any>(`/api/v1/user/events/${email}/${year}`);
  }

  getStarredEvents(year: number): Observable<EventSummary[]> {
    return this.http.get<EventSummary[]>(`/api/v1/user/starred/${year}`);
  }

  starEvent(eventId: string, add: boolean, related: boolean): Observable<any> {
    return this.http.post<any>(`/api/v1/user/star`, { eventId, add, related });
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
}
