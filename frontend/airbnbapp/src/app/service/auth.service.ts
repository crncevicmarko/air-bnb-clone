import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { JwtHelperService } from '@auth0/angular-jwt';
import { BehaviorSubject, Observable, Subject } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private logginSubject: BehaviorSubject<boolean> =
    new BehaviorSubject<boolean>(false);

  private roleSubject: BehaviorSubject<string> = new BehaviorSubject<string>(
    ''
  );

  public isLoggedin: Observable<boolean>;

  public role: Observable<string>;

  constructor(private http: HttpClient, private router: Router) {
    this.isLoggedin = this.logginSubject.asObservable();
    this.role = this.roleSubject.asObservable();
  }

  private headers = new HttpHeaders({ 'Content-Type': 'application/json' });
  jwt: JwtHelperService = new JwtHelperService();

  login(user: any): Observable<any> {
    return this.http.post(
      '/api/users/login',
      { username: user.username, password: user.password },
      { headers: this.headers, responseType: 'json' }
    );
  }

  register(user: any): Observable<any> {
    console.log(user);
    return this.http.post(
      '/api/users/register',
      {
        username: user.username,
        password: user.password,
        role: user.userRole,
        email: user.email,
        firstname: user.firstName,
        lastname: user.lastName,
        location: {
          country: user.country,
          city: user.city,
          streetName: user.streetName,
          streetNumber: user.streetNumber,
        },
      },
      { headers: this.headers, responseType: 'json' }
    );
  }

  logout() {
    localStorage.removeItem('jwt');
    this.checkLoggin();
    this.checkRole();
  }

  checkLoggin(): boolean {
    if (!localStorage.getItem('jwt')) {
      this.setLoggin(false);
      return false;
    } else {
      this.setLoggin(true);
      return true;
    }
  }

  setLoggin(value: boolean) {
    this.logginSubject.next(value);
  }

  setRole(value: string) {
    this.roleSubject.next(value);
  }

  getUserId(): string {
    let token = localStorage.getItem('jwt');
    if (token != null) return this.jwt.decodeToken(token).id;
    else return '';
  }

  getUsername(): string {
    let token = localStorage.getItem('jwt');
    if (token != null) return this.jwt.decodeToken(token).username;
    else return '';
  }

  getRole(): string {
    let token = localStorage.getItem('jwt');
    console.log('Token: ', token);
    if (token != null) return this.jwt.decodeToken(token).role;
    else return '';
  }

  checkRole() {
    let token = localStorage.getItem('jwt');
    if (token != null) this.setRole(this.jwt.decodeToken(token).role);
    else this.setRole('');
  }

  deleteUser(): Observable<any> {
    return this.http.delete('/api/users/delete', {
      headers: this.headers,
      responseType: 'json',
    });
  }

  changeForgottenPassword(
    newPassword: string,
    confirmPassword: string,
    secretCode: string
  ): Observable<any> {
    return this.http.post(
      '/api/users/changeForgottenPassword',
      {
        newPassword: newPassword,
        confirmPassword: confirmPassword,
        code: secretCode,
      },
      { headers: this.headers, responseType: 'json' }
    );
  }

  sendForgottenPasswordEmail(email: string): Observable<any> {
    return this.http.post(
      `${'/api/users/sendforgottemail/'}${email}`,
      {},
      {
        headers: this.headers,
        responseType: 'json',
      }
    );
  }

  changePasswod(
    oldPassword: any,
    newPassword: any,
    confirmPassword: any
  ): Observable<any> {
    return this.http.patch(
      '/api/users/changePassword',
      {
        oldPassword: oldPassword,
        newPassword: newPassword,
        confirmPassword: confirmPassword,
      },
      {
        headers: this.headers,
        responseType: 'json',
      }
    );
  }

  getUserById(id: string): Observable<any> {
    console.log('hostId:', id);
    return this.http.get(`${'/api/users/user/'}${id}`, {
      headers: this.headers,
      responseType: 'json',
    });
  }

  getUserByUsername(username: string): Observable<any> {
    return this.http.get(`${'/api/users/username/'}${username}`, {
      headers: this.headers,
      responseType: 'json',
    });
  }

  sendVerifyingEmail(code: string): Observable<any> {
    return this.http.post(
      `${'/api/users/email/'}${code}`,
      {},
      {
        headers: this.headers,
        responseType: 'json',
      }
    );
  }
}
