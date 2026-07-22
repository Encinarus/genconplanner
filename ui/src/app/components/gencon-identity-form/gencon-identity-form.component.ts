import { Component, model, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-gencon-identity-form',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="d-flex flex-column gap-3 my-2">
      <div *ngIf="showDisplayName()">
        <label class="form-label small text-muted mb-1 fw-bold text-uppercase smaller">Display Name</label>
        <input type="text" 
               class="form-control form-control-sm fw-bold border-primary shadow-sm" 
               [(ngModel)]="displayName" 
               (keyup.enter)="save.emit()" 
               (keyup.esc)="cancel.emit()">
      </div>
      <div>
        <label class="form-label small text-muted mb-1 fw-bold text-uppercase smaller">Gen Con Name</label>
        <input type="text" 
               class="form-control form-control-sm border-primary shadow-sm" 
               [(ngModel)]="genconName" 
               (keyup.enter)="save.emit()" 
               (keyup.esc)="cancel.emit()"
               placeholder="Name on gencon.com">
      </div>
      <div>
        <label class="form-label small text-muted mb-1 fw-bold text-uppercase smaller">Gen Con ID</label>
        <input type="text" 
               class="form-control form-control-sm border-primary shadow-sm" 
               [(ngModel)]="genconId" 
               (keyup.enter)="save.emit()" 
               (keyup.esc)="cancel.emit()"
               placeholder="e.g. 123456">
      </div>
      <div>
        <label class="form-label small text-muted mb-1 fw-bold text-uppercase smaller">Gen Con Email</label>
        <input type="email" 
               class="form-control form-control-sm border-primary shadow-sm" 
               [(ngModel)]="genconEmail" 
               (keyup.enter)="save.emit()" 
               (keyup.esc)="cancel.emit()"
               placeholder="e.g. user@example.com">
      </div>
      <div class="d-flex gap-2 mt-2 pt-2 border-top">
        <button type="button" (click)="save.emit()" class="btn btn-primary btn-sm px-3 shadow-sm flex-grow-1 fw-medium">Save</button>
        <button type="button" (click)="cancel.emit()" class="btn btn-light btn-sm px-3 shadow-sm flex-grow-1 fw-medium">Cancel</button>
      </div>
    </div>
  `
})
export class GenconIdentityFormComponent {
  displayName = model<string>('');
  genconName = model<string>('');
  genconId = model<string>('');
  genconEmail = model<string>('');
  showDisplayName = input<boolean>(true);

  save = output<void>();
  // eslint-disable-next-line @angular-eslint/no-output-native
  cancel = output<void>();
}
