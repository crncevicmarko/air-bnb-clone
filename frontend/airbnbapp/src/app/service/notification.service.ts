import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { Notification1 } from '../model/notification'
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class NotificationService {

  constructor(private http: HttpClient, private router: Router) {}
  private headers = new HttpHeaders({ 'Content-Type': 'application/json' });
  
  getAllByHostId(hostId: any): Observable<any> {
    return this.http.get('/api/notifications/' + hostId,{
      headers: this.headers,
      responseType: 'json',
    });
  }

  createNotification(notification: Notification1): Observable<any> {
    return this.http.post('/api/notifications/create',
    {
      hostId: notification.hostId,
      description: notification.description
    },
    {headers: this.headers, responseType: 'json'})
  }
}

