import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-tier-selector',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="btn-group" [class.btn-group-sm]="size() === 'sm'" [class.btn-group-xs]="size() === 'xs'" role="group">
      <!-- Purchased Tier (Only shown when isGroup is false) -->
      <ng-container *ngIf="!isGroup()">
        <input type="radio" class="btn-check" [name]="namePrefix() + '-' + id()" [id]="namePrefix() + '-purchased-' + id()" 
               autocomplete="off" [checked]="tier() === 'purchased'" (change)="onSelectTier('purchased')">
        <label class="btn btn-outline-warning tier-btn" [class.py-0]="size() === 'xs'" [class.px-1]="size() === 'xs'" [for]="namePrefix() + '-purchased-' + id()" (click)="onLabelClick('purchased', $event)" title="Purchased">
          <i class="bi" [class.bi-ticket-perforated-fill]="tier() === 'purchased'" [class.bi-ticket-perforated]="tier() !== 'purchased'"></i>
        </label>
      </ng-container>

      <!-- Must Have -->
      <input type="radio" class="btn-check" [name]="namePrefix() + '-' + id()" [id]="namePrefix() + '-must-' + id()" 
             autocomplete="off" [checked]="tier() === 'must_have'" (change)="onSelectTier('must_have')">
      <label class="btn btn-outline-danger tier-btn" [class.py-0]="size() === 'xs'" [class.px-1]="size() === 'xs'" [for]="namePrefix() + '-must-' + id()" (click)="onLabelClick('must_have', $event)" [title]="isGroup() ? 'Group Must Have' : 'Must Have'">
        <i class="bi" [class.bi-heart-fill]="tier() === 'must_have'" [class.bi-heart]="tier() !== 'must_have'"></i>
      </label>

      <!-- Very Interested -->
      <input type="radio" class="btn-check" [name]="namePrefix() + '-' + id()" [id]="namePrefix() + '-very-' + id()" 
             autocomplete="off" [checked]="tier() === 'very_interested'" (change)="onSelectTier('very_interested')">
      <label class="btn btn-outline-primary tier-btn" [class.py-0]="size() === 'xs'" [class.px-1]="size() === 'xs'" [for]="namePrefix() + '-very-' + id()" (click)="onLabelClick('very_interested', $event)" [title]="isGroup() ? 'Group Very Interested' : 'Very Interested'">
        <i class="bi" [class.bi-star-fill]="tier() === 'very_interested'" [class.bi-star]="tier() !== 'very_interested'"></i>
      </label>

      <!-- Somewhat Interested -->
      <input type="radio" class="btn-check" [name]="namePrefix() + '-' + id()" [id]="namePrefix() + '-some-' + id()" 
             autocomplete="off" [checked]="tier() === 'somewhat_interested'" (change)="onSelectTier('somewhat_interested')">
      <label class="btn btn-outline-secondary tier-btn" [class.py-0]="size() === 'xs'" [class.px-1]="size() === 'xs'" [for]="namePrefix() + '-some-' + id()" (click)="onLabelClick('somewhat_interested', $event)" [title]="isGroup() ? 'Group Somewhat Interested' : 'Somewhat Interested'">
        <i class="bi" [class.bi-hand-thumbs-up-fill]="tier() === 'somewhat_interested'" [class.bi-hand-thumbs-up]="tier() !== 'somewhat_interested'"></i>
      </label>

      <!-- Not Interested -->
      <input type="radio" class="btn-check" [name]="namePrefix() + '-' + id()" [id]="namePrefix() + '-not-' + id()" 
             autocomplete="off" [checked]="tier() === 'not_interested'" (change)="onSelectTier('not_interested')">
      <label class="btn btn-outline-dark tier-btn" [class.py-0]="size() === 'xs'" [class.px-1]="size() === 'xs'" [for]="namePrefix() + '-not-' + id()" (click)="onLabelClick('not_interested', $event)" [title]="isGroup() ? 'Group Not Interested' : 'Not Interested'">
        <i class="bi bi-slash-circle"></i>
      </label>
    </div>
  `
})
export class TierSelectorComponent {
  id = input.required<string>();
  tier = input<string>('');
  isGroup = input<boolean>(false);
  namePrefix = input<string>('tier');
  size = input<'sm' | 'md' | 'xs'>('sm');

  tierChange = output<string>();
  labelClick = output<string>();

  onSelectTier(newTier: string): void {
    this.tierChange.emit(newTier);
  }

  onLabelClick(clickedTier: string, _event?: Event): void {
    this.labelClick.emit(clickedTier);
  }
}
