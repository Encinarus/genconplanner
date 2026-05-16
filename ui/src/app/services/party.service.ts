import { Injectable, signal, inject, effect } from '@angular/core';
import { ApiService, Party } from './api.service';
import { AuthService } from './auth.service';

@Injectable({
  providedIn: 'root'
})
export class PartyService {
  private api = inject(ApiService);
  private auth = inject(AuthService);

  parties = signal<Party[]>([]);
  loading = signal<boolean>(false);

  constructor() {
    effect(() => {
      const user = this.auth.user();
      if (user) {
        this.fetchParties();
      } else {
        this.parties.set([]);
      }
    });
  }

  fetchParties(): void {
    this.loading.set(true);
    this.api.getParties().subscribe({
      next: (parties) => {
        this.parties.set(parties);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error loading parties:', err);
        this.loading.set(false);
      }
    });
  }

  addParty(party: Party): void {
    this.parties.update(p => [party, ...p]);
  }

  updateParty(party: Party): void {
    this.parties.update(p => p.map(item => item.id === party.id ? party : item));
  }

  removeParty(partyId: number): void {
    this.parties.update(p => p.filter(item => item.id !== partyId));
  }
}
