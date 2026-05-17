import { Injectable, signal } from '@angular/core';
import { initializeApp } from 'firebase/app';
import { getAuth, GoogleAuthProvider, signInWithPopup, signOut, onAuthStateChanged, User } from 'firebase/auth';
import Cookies from 'js-cookie';

const firebaseConfig = {
  apiKey: "AIzaSyAGtjwGiHYFnXE1UbzLTPeIz8Ix06WIdBs",
  authDomain: "genconplanner-v2.firebaseapp.com",
  databaseURL: "https://genconplanner-v2.firebaseio.com",
  projectId: "genconplanner-v2",
  storageBucket: "genconplanner-v2.appspot.com",
  messagingSenderId: "630743534199"
};

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private auth;
  user = signal<User | null>(null);
  displayName = signal<string | null>(null);
  genconName = signal<string | null>(null);
  genconId = signal<string | null>(null);
  genconEmail = signal<string | null>(null);
  authLoaded = signal<boolean>(false);

  constructor() {
    const app = initializeApp(firebaseConfig);
    this.auth = getAuth(app);

    // Initial hint from server if available
    const serverUser = (window as any).serverSideUser;
    if (serverUser) {
      this.user.set(serverUser);
      this.displayName.set(serverUser.displayName || null);
      this.genconName.set(serverUser.genconName || null);
      this.genconId.set(serverUser.genconId || null);
      this.genconEmail.set(serverUser.genconEmail || null);
    }

    onAuthStateChanged(this.auth, (user) => {
      if (user) {
        // Prioritize the name we already have (from server or manual update)
        if (!this.displayName()) {
          this.displayName.set(user.displayName);
        }

        user.getIdToken(true).then(token => {
          Cookies.set('signinToken', token, { path: '/' });
          this.user.set(user);
          this.authLoaded.set(true);

          // Fetch latest user profile from backend
          fetch('/api/v1/user', {
            headers: { 'Authorization': `Bearer ${token}` }
          }).then(res => res.json()).then(data => {
            if (data && data.email) {
              if (data.displayName) this.displayName.set(data.displayName);
              this.genconName.set(data.genconName || null);
              this.genconId.set(data.genconId || null);
              this.genconEmail.set(data.genconEmail || null);
            }
          }).catch(err => console.error('Error fetching user profile', err));
        });
      } else {
        this.user.set(null);
        this.displayName.set(null);
        this.genconName.set(null);
        this.genconId.set(null);
        this.genconEmail.set(null);
        Cookies.remove('signinToken', { path: '/' });
        this.authLoaded.set(true);
      }
    });
  }

  async signIn() {
    const provider = new GoogleAuthProvider();
    try {
      const result = await signInWithPopup(this.auth, provider);
      const token = await result.user.getIdToken(true);
      Cookies.set('signinToken', token, { path: '/' });
      this.user.set(result.user);
      // On sign in, we can trust the firebase name initially
      this.displayName.set(result.user.displayName);
    } catch (error) {
      console.error('Sign in error', error);
    }
  }

  async signOut() {
    try {
      await signOut(this.auth);
      Cookies.remove('signinToken', { path: '/' });
      this.user.set(null);
      this.displayName.set(null);
      this.genconName.set(null);
      this.genconId.set(null);
      this.genconEmail.set(null);
    } catch (error) {
      console.error('Sign out error', error);
    }
  }

  updateUserDisplayName(name: string) {
    this.displayName.set(name);
  }

  updateUserProfile(name: string, genconName: string, genconId: string, genconEmail: string) {
    this.displayName.set(name);
    this.genconName.set(genconName || null);
    this.genconId.set(genconId || null);
    this.genconEmail.set(genconEmail || null);
  }
}
