import { Component, OnInit, signal, inject, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule } from '@angular/router';
import { ApiService, EventSummary } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { StarredService } from '../../services/starred.service';
import { StarButtonComponent } from '../star-button/star-button.component';

@Component({
  selector: 'app-starred',
  standalone: true,
  imports: [CommonModule, RouterModule, StarButtonComponent],
  templateUrl: './starred.component.html',
  styleUrl: './starred.component.css'
})
export class StarredComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private api = inject(ApiService);
  private auth = inject(AuthService);
  private starredService = inject(StarredService);

  year = signal<number>(new Date().getFullYear());
  events = signal<EventSummary[]>([]);
  loading = signal<boolean>(true);
  email = computed(() => this.auth.user()?.email || null);

  constructor() {
    effect(() => {
      if (this.auth.authLoaded()) {
        const year = this.year();
        this.fetchStarred();
      }
    });
  }

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      this.year.set(+params['year'] || new Date().getFullYear());
      this.starredService.fetchStarred(this.year());
    });
  }

  fetchStarred(): void {
    this.loading.set(true);
    this.api.getStarredEvents(this.year()).subscribe({
      next: (data) => {
        this.events.set(data);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error fetching starred events', err);
        this.loading.set(false);
      }
    });
  }
}
