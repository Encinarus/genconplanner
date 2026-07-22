import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { EventSummary } from '../../services/api.service';

@Component({
  selector: 'app-ticket-availability-badge',
  standalone: true,
  imports: [CommonModule],
  template: `
    <ul class="list-inline eventTickets mb-0 small" *ngIf="event() as e">
      <li class="list-inline-item" *ngIf="e.wedEvents > 0" [class.noTickets]="e.wedTickets === 0"><strong>Wed</strong> {{ e.wedTickets }}</li>
      <li class="list-inline-item" *ngIf="e.thuEvents > 0" [class.noTickets]="e.thuTickets === 0"><strong>Thu</strong> {{ e.thuTickets }}</li>
      <li class="list-inline-item" *ngIf="e.friEvents > 0" [class.noTickets]="e.friTickets === 0"><strong>Fri</strong> {{ e.friTickets }}</li>
      <li class="list-inline-item" *ngIf="e.satEvents > 0" [class.noTickets]="e.satTickets === 0"><strong>Sat</strong> {{ e.satTickets }}</li>
      <li class="list-inline-item" *ngIf="e.sunEvents > 0" [class.noTickets]="e.sunTickets === 0"><strong>Sun</strong> {{ e.sunTickets }}</li>
    </ul>
  `
})
export class TicketAvailabilityBadgeComponent {
  event = input.required<EventSummary>();
}
