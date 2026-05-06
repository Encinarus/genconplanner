import { Component, OnInit, signal, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, Event } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { StarredService } from '../../services/starred.service';
import { StarButtonComponent } from '../star-button/star-button.component';

@Component({
  selector: 'app-event-detail',
  standalone: true,
  imports: [CommonModule, RouterModule, StarButtonComponent],
  templateUrl: './event-detail.component.html',
  styleUrl: './event-detail.component.css'
})
export class EventDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private api = inject(ApiService);
  private starredService = inject(StarredService);

  eventId = signal<string>('');
  event = signal<Event | null>(null);
  loading = signal<boolean>(true);
  
  groupedEvents = computed(() => {
    const e = this.event();
    if (!e || !e.relatedEvents) return [];

    const groups: { [key: string]: any[] } = {};
    e.relatedEvents.forEach(rel => {
      // Use the raw date string to determine the day
      const date = new Date(rel.startTime);
      const day = date.toLocaleDateString('en-US', { weekday: 'long' });
      if (!groups[day]) {
        groups[day] = [];
      }
      groups[day].push(rel);
    });

    // Sort events within each day by time
    Object.keys(groups).forEach(day => {
      groups[day].sort((a, b) => new Date(a.startTime).getTime() - new Date(b.startTime).getTime());
    });

    // Sort days: Wed, Thu, Fri, Sat, Sun
    const dayOrder = ['Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
    return dayOrder
      .filter(day => groups[day])
      .map(day => ({
        day,
        events: groups[day]
      }));
  });

  private auth = inject(AuthService);

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      this.eventId.set(params['eid']);
      this.fetchEvent();
    });
  }

  fetchEvent(): void {
    this.loading.set(true);
    this.api.getEvent(this.eventId()).subscribe({
      next: (data) => {
        this.event.set(data);
        this.starredService.fetchStarred(data.year);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error fetching event', err);
        this.loading.set(false);
      }
    });
  }

  isSessionStarred(sid: string): boolean {
    return this.starredService.isStarred(sid);
  }
}
