import { Injectable, signal } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class PartyStreamService {
  private eventSource!: EventSource;
  public latestInterestUpdate = signal<any | null>(null);

  connect(partyId: number | string) {
    this.disconnect();
    this.eventSource = new EventSource(`/api/v1/party/${partyId}/stream`);

    this.eventSource.addEventListener('interest_update', (event: any) => {
      try {
        const data = JSON.parse(event.data);
        console.log('Real-time update received from server:', data);
        this.latestInterestUpdate.set(data);
      } catch (e) {
        console.error('Error parsing SSE message:', e);
      }
    });

    this.eventSource.onerror = (error) => {
      console.warn('SSE stream error, browser will automatically reconnect...', error);
    };
  }

  disconnect() {
    if (this.eventSource) {
      this.eventSource.close();
    }
  }
}
