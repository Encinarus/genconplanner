import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class LinkService {
  // Central place to manage the UI prefix for the new Angular application.
  // Change this to '' when the new UI becomes the main UI at the root.
  private readonly uiPrefix = '/v2';

  /**
   * Returns a full URL path including the UI prefix.
   * Useful for window.open or <a> tags without routerLink.
   */
  getEventUrl(eventId: string): string {
    return `${this.uiPrefix}/event/${eventId}`;
  }

  /**
   * Returns a router link array for [routerLink] directive.
   * If the base href is adjusted later, this can be updated accordingly.
   */
  getEventRouterLink(eventId: string): any[] {
    return ['/event', eventId];
  }

  getCategoryRouterLink(year: number, categoryCode: string): any[] {
    return ['/cat', year, categoryCode];
  }

  getSearchRouterLink(): any[] {
    return ['/search/by_system'];
  }

  getHomeRouterLink(): any[] {
    return ['/'];
  }

  getStarredRouterLink(year: number): any[] {
    return ['/starred', year];
  }

  getAboutRouterLink(): any[] {
    return ['/about'];
  }
}
