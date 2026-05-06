import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService, Category } from '../../services/api.service';
import { RouterModule, ActivatedRoute } from '@angular/router';
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'app-category-list',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './category-list.component.html',
  styleUrl: './category-list.component.css'
})
export class CategoryListComponent implements OnInit {
  private api = inject(ApiService);
  private route = inject(ActivatedRoute);
  private titleService = inject(Title);
  
  categories = signal<Category[]>([]);
  year = signal<number>(new Date().getFullYear());
  loading = signal<boolean>(true);

  constructor() {
    this.titleService.setTitle('Categories');
  }

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      const yearParam = params['year'];
      if (yearParam) {
        this.year.set(+yearParam);
      }
      this.fetchCategories();
    });
  }

  fetchCategories(): void {
    this.loading.set(true);
    this.api.getCategories(this.year()).subscribe({
      next: (data) => {
        this.categories.set(data);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error fetching categories', err);
        this.loading.set(false);
      }
    });
  }
}
