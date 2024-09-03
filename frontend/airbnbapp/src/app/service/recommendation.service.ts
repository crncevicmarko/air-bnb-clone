import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { Observable } from 'rxjs';
import { Accommodation } from '../model/accommodation';
import { JwtHelperService } from '@auth0/angular-jwt';

@Injectable({
  providedIn: 'root',
})
export class RecommendationService {
  constructor(private http: HttpClient, private router: Router) {}

  private headers = new HttpHeaders({ 'Content-Type': 'application/json' });
  jwt: JwtHelperService = new JwtHelperService();

  insert(accommodation: Accommodation): Observable<any> {
    let username;
    let token = localStorage.getItem('jwt');
    if (token != null) {
      username = this.jwt.decodeToken(token).username;
    }

    return this.http.post(
      '/api/recommend/insert',
      {
        username: username,
        accomoId: accommodation._id,
      },
      { headers: this.headers, responseType: 'json' }
    );
  }

  getAllRecommendations(): Observable<any> {
    let username;
    let token = localStorage.getItem('jwt');
    if (token != null) {
      username = this.jwt.decodeToken(token).username;
    }

    return this.http.get('/api/recommend/' + username, {
      headers: this.headers,
      responseType: 'json',
    });
  }
}
