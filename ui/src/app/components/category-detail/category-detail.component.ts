import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { EventCatalogViewComponent } from '../event-catalog-view/event-catalog-view.component';

@Component({
  selector: 'app-category-detail',
  standalone: true,
  imports: [CommonModule, EventCatalogViewComponent],
  templateUrl: './category-detail.component.html',
  styleUrl: './category-detail.component.css'
})
export class CategoryDetailComponent {}
