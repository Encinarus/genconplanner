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
  authLoaded = signal<boolean>(false);

  constructor() {
    const app = initializeApp(firebaseConfig);
    this.auth = getAuth(app);

    // Initial hint from server if available
    const serverUser = (window as any).serverSideUser;
    if (serverUser) {
      this.user.set(serverUser);
      this.displayName.set(serverUser.displayName);
    }

    onAuthStateChanged(this.auth, (user) => {
      this.user.set(user);
      if (user) {
        // Prioritize the name we already have (from server or manual update)
        if (!this.displayName()) {
          this.displayName.set(user.displayName);
        }

        user.getIdToken(true).then(token => {
          Cookies.set('signinToken', token, { path: '/' });
          this.authLoaded.set(true);
        });
      } else {
        this.displayName.set(null);
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
    } catch (error) {
      console.error('Sign out error', error);
    }
  }

  updateUserDisplayName(name: string) {
    this.displayName.set(name);
  }
}
