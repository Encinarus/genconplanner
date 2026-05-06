import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, EventSummary } from '../../services/api.service';

@Component({
  selector: 'app-search',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './search.component.html',
  styleUrl: './search.component.css'
})
export class SearchComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private api = inject(ApiService);

  year = signal<number>(new Date().getFullYear());
  query = signal<string>('');
  events = signal<EventSummary[]>([]);
  loading = signal<boolean>(true);

  ngOnInit(): void {
    this.route.queryParams.subscribe(params => {
      this.query.set(params['q'] || '');
      this.year.set(+(params['year'] || new Date().getFullYear()));
      this.fetchResults();
    });
  }

  fetchResults(): void {
    if (!this.query()) {
      this.events.set([]);
      this.loading.set(false);
      return;
    }
    this.loading.set(true);
    this.api.searchEvents({ year: this.year(), search: this.query() }).subscribe({
      next: (data) => {
        this.events.set(data);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error searching events', err);
        this.loading.set(false);
      }
    });
  }
}
