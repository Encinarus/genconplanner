import { Injectable, inject, signal, effect } from '@angular/core';
import { ApiService, StarredPageData } from './api.service';
import { AuthService } from './auth.service';
import { tap } from 'rxjs/operators';

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
        // If we have a pending star, execute it
        if (this.pendingStar) {
          const { eventId, year, starGroup } = this.pendingStar;
          this.pendingStar = null;
          // Small delay to ensure auth state is fully propagated
          setTimeout(() => this.toggleStar(eventId, year, starGroup), 500);
          return;
        }

        // If a year was requested but we haven't loaded it for this user yet
        const year = this.loadedYear();
        if (year > 0 && !this.starredPageData()) {
          this.fetchStarred(year);
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

  unstarGroup(eventId: string, year: number): void {
    const user = this.auth.user();
    if (!user) return;

    const currentData = this.starredPageData();
    if (currentData) {
      const target = currentData.individualEvents.find(e => e.eventId === eventId);
      if (target) {
        const updatedEvents = currentData.individualEvents.filter(e => !(e.categoryCode === target.categoryCode && e.title === target.title && e.shortDescription === target.shortDescription));
        this.starredPageData.set({ ...currentData, individualEvents: updatedEvents });

        const groupEventIds = currentData.individualEvents.filter(e => e.categoryCode === target.categoryCode && e.title === target.title && e.shortDescription === target.shortDescription).map(e => e.eventId);
        this.starredIds.update(ids => ids.filter(id => !groupEventIds.includes(id)));
        this.groupStarredIds.update(ids => ids.filter(id => !groupEventIds.includes(id)));
      }
    }

    this.api.starEvent(eventId, false, true, '', true).subscribe({
      next: () => {
        this.fetchStarred(year, true);
      },
      error: (err) => {
        console.error('Error unstarring group', err);
        this.fetchStarred(year, true);
      }
    });
  }

  updateTier(eventId: string, year: number, tier: string, starGroup: boolean = false): void {
    const user = this.auth.user();
    if (!user) return;

    // Optimistically update the tier in local state if possible
    const currentData = this.starredPageData();
    if (currentData) {
      if (starGroup) {
        const target = currentData.individualEvents.find(e => e.eventId === eventId);
        if (target) {
          const updatedEvents = currentData.individualEvents.map(e => {
            const isSameGroup = e.categoryCode === target.categoryCode && e.title === target.title && e.shortDescription === target.shortDescription;
            if (isSameGroup) {
              return {
                ...e,
                groupTier: tier,
                tier: e.isOverride ? e.tier : tier
              };
            }
            return e;
          });
          this.starredPageData.set({ ...currentData, individualEvents: updatedEvents });
        }
      } else {
        const updatedEvents = currentData.individualEvents.map(e => 
          e.eventId === eventId ? { ...e, tier, isOverride: true } : e
        );
        this.starredPageData.set({ ...currentData, individualEvents: updatedEvents });
      }
    }

    this.api.starEvent(eventId, true, starGroup, tier).subscribe({
      next: () => {
        this.fetchStarred(year, true);
      },
      error: (err) => {
        console.error('Error updating tier', err);
        this.fetchStarred(year, true);
      }
    });
  }

  removeOverride(eventId: string, year: number): void {
    const user = this.auth.user();
    if (!user) return;

    const currentData = this.starredPageData();
    if (currentData) {
      const updatedEvents = currentData.individualEvents.map(e => 
        e.eventId === eventId ? { ...e, isOverride: false, tier: e.groupTier || 'not_interested' } : e
      );
      this.starredPageData.set({ ...currentData, individualEvents: updatedEvents });
    }

    this.api.starEvent(eventId, false, false, '').subscribe({
      next: () => {
        this.fetchStarred(year, true);
      },
      error: (err) => {
        console.error('Error removing override', err);
        this.fetchStarred(year, true);
      }
    });
  }

  removeGroupDefault(eventId: string, year: number): void {
    const user = this.auth.user();
    if (!user) return;

    const currentData = this.starredPageData();
    if (currentData) {
      const target = currentData.individualEvents.find(e => e.eventId === eventId);
      if (target) {
        const groupEvents = currentData.individualEvents.filter(e => e.categoryCode === target.categoryCode && e.title === target.title && e.shortDescription === target.shortDescription);
        const override = groupEvents.find(e => e.isOverride);
        
        let updatedEvents;
        if (override) {
          updatedEvents = currentData.individualEvents.map(e => {
            const isSameGroup = e.categoryCode === target.categoryCode && e.title === target.title && e.shortDescription === target.shortDescription;
            if (isSameGroup) {
              return {
                ...e,
                groupTier: override.tier,
                isOverride: e.eventId === override.eventId ? false : e.isOverride,
                tier: e.eventId === override.eventId ? override.tier : (e.isOverride ? e.tier : override.tier)
              };
            }
            return e;
          });
        } else {
          updatedEvents = currentData.individualEvents.map(e => {
            const isSameGroup = e.categoryCode === target.categoryCode && e.title === target.title && e.shortDescription === target.shortDescription;
            if (isSameGroup) {
              return {
                ...e,
                groupTier: '',
                tier: 'not_interested'
              };
            }
            return e;
          });
        }
        this.starredPageData.set({ ...currentData, individualEvents: updatedEvents });
      }
    }

    this.api.starEvent(eventId, false, true, '').subscribe({
      next: () => {
        this.fetchStarred(year, true);
      },
      error: (err) => {
        console.error('Error removing group default', err);
        this.fetchStarred(year, true);
      }
    });
  }

  bulkClear(year: number) {
    return this.api.bulkClearStarred(year).pipe(
      tap(() => this.fetchStarred(year, true))
    );
  }

  bulkReplace(year: number, text: string, overwrite: boolean, asGroups: boolean) {
    return this.api.bulkReplaceStarred(year, text, overwrite, asGroups).pipe(
      tap(() => this.fetchStarred(year, true))
    );
  }
}
