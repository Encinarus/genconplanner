import { Component, input, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { StarredService } from '../../services/starred.service';

@Component({
  selector: 'app-star-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    <button type="button"
            class="btn btn-light btn-md border sm-star"
            [class.active]="isStarred()"
            (click)="$event.stopPropagation(); toggleStar()">
        <span class="material-icons" [style.font-size]="iconSize()">
            {{ isStarred() ? 'star' : 'star_outline' }}
        </span>
    </button>
  `,
  styles: [`
    .sm-star {
        padding: 0.1rem 0.4rem;
        color: #6c757d;
        background: white;
        transition: all 0.2s;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .sm-star:hover {
        color: #ffc107;
        background: #fff9e6;
    }

    .sm-star.active {
        color: #ffc107 !important;
        border-color: #ffc107 !important;
    }
  `]
})
export class StarButtonComponent {
  private starredService = inject(StarredService);

  eventId = input.required<string>();
  year = input.required<number>();
  isGroup = input<boolean>(true);
  iconSize = input<string>('1.2rem');

  isStarred = computed(() => this.starredService.isStarred(this.eventId()));

  toggleStar(): void {
    this.starredService.toggleStar(this.eventId(), this.year(), this.isGroup());
  }
}
