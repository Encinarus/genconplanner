import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService, Category } from '../../services/api.service';
import { RouterModule } from '@angular/router';

@Component({
  selector: 'app-category-list',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './category-list.component.html',
  styleUrl: './category-list.component.css'
})
export class CategoryListComponent implements OnInit {
  private api = inject(ApiService);
  
  categories = signal<Category[]>([]);
  year = signal<number>(new Date().getFullYear());
  loading = signal<boolean>(true);

  ngOnInit(): void {
    console.log('CategoryListComponent: Initializing for year', this.year());
    
    this.api.getCategories(this.year()).subscribe({
      next: (data) => {
        console.log('CategoryListComponent: Received categories', data.length);
        this.categories.set(data);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('CategoryListComponent: Error fetching categories', err);
        this.loading.set(false);
      }
    });
  }
}
