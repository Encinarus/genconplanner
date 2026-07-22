import { Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';

export interface BggGameSystem {
  name?: string;
  systemName?: string;
  bggId?: number;
  bggRating?: number;
  numBggRatings?: number;
  yearPublished?: number;
}

@Component({
  selector: 'app-bgg-link',
  standalone: true,
  imports: [CommonModule],
  template: `
    <ng-container *ngIf="gameSystem() as sys">
      <ng-container *ngIf="sys.bggId; else noBgg">
        <a [href]="'https://boardgamegeek.com/boardgame/' + sys.bggId" target="_blank" rel="noopener noreferrer" [class]="linkClass()" class="text-decoration-none d-inline-flex align-items-center">
          <span>{{ sys.systemName || sys.name }}</span>
          <small class="text-muted" [style.font-size]="smallFontSize()" style="margin-left: 0.25rem;">
            <i class="bi bi-link-45deg"></i> BGG<ng-container *ngIf="(sys.numBggRatings || 0) >= 100"> {{ sys.bggRating!.toFixed(1) }}, {{ sys.numBggRatings }} ratings</ng-container><ng-container *ngIf="showYear() && sys.yearPublished"> ({{ sys.yearPublished }})</ng-container>
          </small>
        </a>
      </ng-container>
      <ng-template #noBgg>
        <span [class]="textClass()">{{ sys.systemName || sys.name }}</span>
      </ng-template>
    </ng-container>
  `
})
export class BggLinkComponent {
  gameSystem = input.required<BggGameSystem>();
  linkClass = input<string>('text-dark');
  textClass = input<string>('');
  smallFontSize = input<string>('0.85rem');
  showYear = input<boolean>(true);
}
