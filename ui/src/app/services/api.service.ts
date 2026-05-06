import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
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
}

export interface Event {
  eventId: string;
  year: number;
  active: boolean;
  title: string;
  shortDescription: string;
  longDescription: string;
  categoryCode: string;
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
    let queryParams = `?year=${params.year}`;
    if (params.cat) queryParams += `&cat=${params.cat}`;
    if (params.search) queryParams += `&search=${params.search}`;
    return this.http.get<EventSummary[]>(`/api/v1/events${queryParams}`);
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
}
