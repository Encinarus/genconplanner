import { Injectable, inject, signal, effect } from '@angular/core';
import { ApiService } from './api.service';
import { AuthService } from './auth.service';

@Injectable({
  providedIn: 'root'
})
export class StarredService {
  private api = inject(ApiService);
  private auth = inject(AuthService);

  starredIds = signal<string[]>([]);
  groupStarredIds = signal<string[]>([]);
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
        this.starredIds.set([]);
        this.groupStarredIds.set([]);
      }
    });
  }

  fetchStarred(year: number): void {
    const user = this.auth.user();
    if (!user || !user.email) {
      this.loadedYear.set(year);
      return;
    }

    this.loadedYear.set(year);
    this.api.getUserEvents(user.email, year).subscribe(data => {
      this.updateState(data.starredClusters || [], data.starredEvents || []);
    });
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

    // Optimistic update
    const allToUpdate = starGroup ? [eventId, ...relatedEventIds] : [eventId];
    
    if (newStarred) {
      this.starredIds.update(ids => [...new Set([...ids, ...allToUpdate])]);
      if (starGroup) {
        this.groupStarredIds.update(ids => [...new Set([...ids, ...allToUpdate])]);
      }
    } else {
      this.starredIds.update(ids => ids.filter(id => !allToUpdate.includes(id)));
      // If unstarring, it's definitely no longer a group star for these IDs
      // If it was an individual unstar (starGroup=false), we should also clear the group status 
      // for any related IDs because the group is now broken.
      const idsToClearGroup = starGroup ? allToUpdate : [eventId, ...relatedEventIds];
      this.groupStarredIds.update(ids => ids.filter(id => !idsToClearGroup.includes(id)));
    }

    this.api.starEvent(eventId, newStarred, starGroup).subscribe({
      next: () => {
        this.fetchStarred(year);
      },
      error: (err) => {
        console.error('Error starring event', err);
        // Rollback
        this.fetchStarred(year);
      }
    });
  }
}
