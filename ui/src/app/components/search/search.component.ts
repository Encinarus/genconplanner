import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { EventCatalogViewComponent } from '../event-catalog-view/event-catalog-view.component';

@Component({
  selector: 'app-search',
  standalone: true,
  imports: [CommonModule, EventCatalogViewComponent],
  templateUrl: './search.component.html',
  styleUrl: './search.component.css'
})
export class SearchComponent {}
