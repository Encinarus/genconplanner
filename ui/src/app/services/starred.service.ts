import { Injectable, inject, signal, effect } from '@angular/core';
import { ApiService, StarredPageData } from './api.service';
import { AuthService } from './auth.service';

@Injectable({
  providedIn: 'root'
})
export class StarredService {
  private api = inject(ApiService);
  private auth = inject(AuthService);

  starredIds = signal<string[]>([]);
  groupStarredIds = signal<string[]>([]);
  starredPageData = signal<StarredPageData | null>(null);

  private loadedYear = signal<number>(0);
  private pendingStar: { eventId: string, year: number, starGroup: boolean } | null = null;

  constructor() {
    // Re-fetch when user changes and handle pending actions
    effect(() => {
      const user = this.auth.user();
      if (user) {
        // If we have a year loaded or pending, fetch it
        const yearToFetch = this.loadedYear() || (this.pendingStar ? this.pendingStar.year : 0);
        if (yearToFetch > 0) {
          this.fetchStarred(yearToFetch);
          
          // Execute pending star if exists
          if (this.pendingStar) {
            const { eventId, year, starGroup } = this.pendingStar;
            this.pendingStar = null;
            setTimeout(() => this.toggleStar(eventId, year, starGroup), 500);
          }
        }
      } else {
        this.clearCache();
      }
    });
  }

  private getCacheKey(year: number): string {
    const user = this.auth.user();
    if (!user || !user.email) return '';
    return `starred_events_${user.email}_${year}`;
  }

  private loadFromCache(year: number): void {
    const key = this.getCacheKey(year);
    if (!key) return;

    try {
      const cached = localStorage.getItem(key);
      if (cached) {
        const data: StarredPageData = JSON.parse(cached);
        this.starredPageData.set(data);
        this.updateIdsFromData(data);
      }
    } catch (e) {
      console.error('Error loading starred cache', e);
    }
  }

  private saveToCache(year: number, data: StarredPageData): void {
    const key = this.getCacheKey(year);
    if (!key) return;

    try {
      localStorage.setItem(key, JSON.stringify(data));
    } catch (e) {
      console.error('Error saving starred cache', e);
    }
  }

  private clearCache(): void {
    this.starredIds.set([]);
    this.groupStarredIds.set([]);
    this.starredPageData.set(null);
    // Note: We don't necessarily want to wipe localStorage for all years,
    // but the current user's state in memory should be cleared.
  }

  fetchStarred(year: number, skipCache: boolean = false): void {
    const user = this.auth.user();
    if (!user || !user.email) {
      this.loadedYear.set(year);
      return;
    }

    this.loadedYear.set(year);

    // 1. Immediate load from cache (if requested)
    if (!skipCache) {
      this.loadFromCache(year);
    }

    // 2. Background refresh
    this.api.getStarredPageData(year).subscribe(data => {
      this.starredPageData.set(data);
      this.updateIdsFromData(data);
      this.saveToCache(year, data);
    });
  }

  private updateIdsFromData(data: StarredPageData): void {
    this.groupStarredIds.set(data.starredClusters || []);
    this.starredIds.set([...new Set([...(data.starredClusters || []), ...(data.starredEvents || [])])]);
  }

  updateState(starredClusters: string[], starredEvents: string[]): void {
    this.groupStarredIds.set(starredClusters);
    this.starredIds.set([...new Set([...starredClusters, ...starredEvents])]);
  }

  isStarred(eventId: string): boolean {
    return this.starredIds().includes(eventId);
  }

  isGroupStarred(eventId: string): boolean {
    return this.groupStarredIds().includes(eventId);
  }

  toggleStar(eventId: string, year: number, starGroup: boolean = true, relatedEventIds: string[] = []): void {
    const user = this.auth.user();
    if (!user) {
      this.pendingStar = { eventId, year, starGroup };
      this.auth.signIn();
      return;
    }

    const isCurrentlyStarred = starGroup ? this.isGroupStarred(eventId) : this.isStarred(eventId);
    const newStarred = !isCurrentlyStarred;

    // Optimistic update of IDs
    const allToUpdate = starGroup ? [eventId, ...relatedEventIds] : [eventId];
    
    if (newStarred) {
      this.starredIds.update(ids => [...new Set([...ids, ...allToUpdate])]);
      if (starGroup) {
        this.groupStarredIds.update(ids => [...new Set([...ids, ...allToUpdate])]);
      }
    } else {
      this.starredIds.update(ids => ids.filter(id => !allToUpdate.includes(id)));
      const idsToClearGroup = starGroup ? allToUpdate : [eventId, ...relatedEventIds];
      this.groupStarredIds.update(ids => ids.filter(id => !idsToClearGroup.includes(id)));
    }

    this.api.starEvent(eventId, newStarred, starGroup).subscribe({
      next: () => {
        // Refresh full data in background to update cache and calendar
        // We skip cache here to avoid flickering back to stale data
        this.fetchStarred(year, true);
      },
      error: (err) => {
        console.error('Error starring event', err);
        // Rollback
        this.fetchStarred(year, true);
      }
    });
  }
}
