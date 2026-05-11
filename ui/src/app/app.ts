import { Component, signal, OnInit, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { NavbarComponent } from './components/navbar/navbar.component';
import { ApiService } from './services/api.service';
import { DatePipe } from '@angular/common';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, NavbarComponent, DatePipe],
  templateUrl: './app.html',
  styleUrl: './app.css'
})
export class App implements OnInit {
  protected readonly title = signal('ui');
  protected readonly lastUpdate = signal<string | null>(null);
  private api = inject(ApiService);

  ngOnInit() {
    this.api.getLastUpdate().subscribe(data => {
      this.lastUpdate.set(data.lastUpdate);
    });
  }
}
