import { Component, signal, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-navbar',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule],
  templateUrl: './navbar.component.html',
  styleUrl: './navbar.component.css'
})
export class NavbarComponent {
  private router = inject(Router);
  private authService = inject(AuthService);
  
  year = signal<number>(new Date().getFullYear());
  displayName = computed(() => this.authService.user()?.displayName || null);
  searchQuery = signal<string>('');

  onSearch(event: Event) {
    event.preventDefault();
    if (this.searchQuery().trim()) {
      this.router.navigate(['/search/by_system'], { queryParams: { q: this.searchQuery(), year: this.year() } });
    }
  }

  popupSignIn() {
    this.authService.signIn();
  }

  signOut() {
    this.authService.signOut();
  }
}
