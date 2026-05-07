import { Component, input, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { StarredService } from '../../services/starred.service';

@Component({
  selector: 'app-star-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    <button type="button"
            class="btn btn-light border star-btn"
            [class.active]="isStarred()"
            [style.width]="btnSize()"
            [style.height]="btnSize()"
            (click)="$event.stopPropagation(); toggleStar()">
        <span class="material-icons" [style.font-size]="iconSize()">
            {{ isStarred() ? 'star' : 'star_outline' }}
        </span>
    </button>
  `,
  styles: [`
    .star-btn {
        padding: 0;
        color: #6c757d;
        background: white;
        transition: all 0.2s;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        line-height: 1;
    }

    .star-btn:hover {
        color: #333;
        background: #f8f9fa;
        border-color: #333 !important;
    }

    .star-btn.active {
        color: #000 !important;
        border-color: #000 !important;
        background: #f0f0f0;
    }
  `]
})
export class StarButtonComponent {
  private starredService = inject(StarredService);

  eventId = input.required<string>();
  year = input.required<number>();
  isGroup = input<boolean>(true);
  isGroupMode = input<boolean>(false);
  relatedEventIds = input<string[]>([]);
  iconSize = input<string>('1.2rem');
  
  btnSize = computed(() => {
    const iconSize = this.iconSize();
    if (iconSize.endsWith('rem')) {
      return (parseFloat(iconSize) * 1.6) + 'rem';
    }
    return '2.2rem'; // Fallback
  });

  isStarred = computed(() => {
    if (this.isGroupMode()) {
      return this.starredService.isGroupStarred(this.eventId());
    }
    return this.starredService.isStarred(this.eventId());
  });

  toggleStar(): void {
    this.starredService.toggleStar(this.eventId(), this.year(), this.isGroup(), this.relatedEventIds());
  }
}
