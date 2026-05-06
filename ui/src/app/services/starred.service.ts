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
  private loadedYear = signal<number>(0);

  constructor() {
    // Re-fetch when user changes
    effect(() => {
      const user = this.auth.user();
      if (user && this.loadedYear() > 0) {
        this.fetchStarred(this.loadedYear());
      } else if (!user) {
        this.starredIds.set([]);
      }
    });
  }

  fetchStarred(year: number): void {
    const user = this.auth.user();
    if (!user || !user.email) return;

    this.loadedYear.set(year);
    this.api.getUserEvents(user.email, year).subscribe(data => {
      const allStarred = [...(data.starredEvents || []), ...(data.starredClusters || [])];
      this.starredIds.set(allStarred);
    });
  }

  isStarred(eventId: string): boolean {
    return this.starredIds().includes(eventId);
  }

  toggleStar(eventId: string, year: number, starGroup: boolean = true): void {
    const user = this.auth.user();
    if (!user) {
      this.auth.signIn();
      return;
    }

    const isCurrentlyStarred = this.isStarred(eventId);
    const newStarred = !isCurrentlyStarred;

    // Optimistic update
    if (newStarred) {
      this.starredIds.update(ids => [...ids, eventId]);
    } else {
      this.starredIds.update(ids => ids.filter(id => id !== eventId));
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
