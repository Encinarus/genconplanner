import { Injectable, signal } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class PartyStreamService {
  private eventSource!: EventSource;
  public latestInterestUpdate = signal<any | null>(null);
  public streamResumed = signal<number>(0);

  private activePartyId: number | string | null = null;
  private visibilityHandler?: () => void;

  connect(partyId: number | string) {
    this.activePartyId = partyId;
    this.initStream(partyId);

    if (!this.visibilityHandler) {
      this.visibilityHandler = () => {
        if (document.visibilityState === 'hidden') {
          console.log('Page hidden: Suspending SSE stream to conserve Heroku connections.');
          if (this.eventSource) {
            this.eventSource.close();
          }
        } else if (document.visibilityState === 'visible' && this.activePartyId) {
          console.log('Page visible: Resuming SSE stream and requesting fresh state.');
          this.initStream(this.activePartyId);
          this.streamResumed.update(c => c + 1);
        }
      };
      document.addEventListener('visibilitychange', this.visibilityHandler);
    }
  }

  private initStream(partyId: number | string) {
    if (this.eventSource) {
      this.eventSource.close();
    }
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
    this.activePartyId = null;
    if (this.eventSource) {
      this.eventSource.close();
    }
    if (this.visibilityHandler) {
      document.removeEventListener('visibilitychange', this.visibilityHandler);
      this.visibilityHandler = undefined;
    }
  }
}
