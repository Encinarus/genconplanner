import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, EventSummary } from '../../services/api.service';

@Component({
  selector: 'app-category-detail',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './category-detail.component.html',
  styleUrl: './category-detail.component.css'
})
export class CategoryDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private api = inject(ApiService);

  year = signal<number>(0);
  categoryCode = signal<string>('');
  events = signal<EventSummary[]>([]);
  loading = signal<boolean>(true);

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      this.year.set(+params['year']);
      this.categoryCode.set(params['cat']);
      this.fetchEvents();
    });
  }

  fetchEvents(): void {
    this.loading.set(true);
    this.api.searchEvents({ year: this.year(), cat: this.categoryCode() }).subscribe({
      next: (data) => {
        this.events.set(data);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error fetching events', err);
        this.loading.set(false);
      }
    });
  }
}
