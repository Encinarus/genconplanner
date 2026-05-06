import { Component } from '@angular/core';

@Component({
  selector: 'app-about',
  standalone: true,
  template: `
    <h1 class="pb-2 pt-4 mt-4 mb-3 border-bottom">About Gen Con Planner</h1>
    <p>This is a tool to help you find and star events for Gen Con.</p>
    <p>The new Angular UI is currently under development!</p>
  `,
  styles: [`
    :host { display: block; padding: 20px; }
  `]
})
export class AboutComponent {}
