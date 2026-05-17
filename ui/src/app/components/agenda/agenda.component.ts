import { Component, Input, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { ApiService, StarredEventDetail } from '../../services/api.service';
import { LinkService } from '../../services/link.service';

@Component({
  selector: 'app-agenda',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './agenda.component.html',
  styleUrl: './agenda.component.css'
})
export class AgendaComponent implements OnInit {
  @Input({ required: true }) year!: number;

  private api = inject(ApiService);
  public linkService = inject(LinkService);

  agendaItems = signal<StarredEventDetail[]>([]);
  loading = signal<boolean>(true);

  ngOnInit(): void {
    this.loadAgenda();
  }

  loadAgenda(): void {
    this.loading.set(true);
    this.api.getAgenda(this.year).subscribe({
      next: (items) => {
        this.agendaItems.set(items);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error loading agenda:', err);
        this.loading.set(false);
      }
    });
  }

  formatTiming(start: string, end: string): string {
    const s = new Date(start);
    const e = new Date(end);
    const timeZone = 'America/Indiana/Indianapolis';
    const options: Intl.DateTimeFormatOptions = { 
      timeZone,
      weekday: 'short', 
      hour: 'numeric', 
      minute: '2-digit',
      hour12: true 
    };
    const timeOptions: Intl.DateTimeFormatOptions = {
      timeZone,
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    };
    return `${s.toLocaleDateString('en-US', options)} - ${e.toLocaleTimeString('en-US', timeOptions)}`;
  }

  getTierIconClass(tier: string): string {
    switch (tier) {
      case 'purchased': return 'bi-ticket-perforated-fill text-warning';
      case 'must_have': return 'bi-heart-fill text-danger';
      case 'very_interested': return 'bi-star-fill text-primary';
      case 'somewhat_interested': return 'bi-hand-thumbs-up-fill text-secondary';
      default: return 'bi-star-fill text-primary';
    }
  }

  getTierClass(tier: string): string {
    switch (tier) {
      case 'purchased': return 'tier-purchased';
      case 'must_have': return 'tier-must';
      case 'very_interested': return 'tier-very';
      case 'somewhat_interested': return 'tier-some';
      default: return '';
    }
  }
}
