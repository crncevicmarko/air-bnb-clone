import { formatDate } from '@angular/common';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { JwtHelperService } from '@auth0/angular-jwt';
import { NgbDate } from '@ng-bootstrap/ng-bootstrap';
import { end, start } from '@popperjs/core';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class ReservationService {
  constructor(private http: HttpClient, private router: Router) {}

  private headers = new HttpHeaders({ 'Content-Type': 'application/json' });
  jwt: JwtHelperService = new JwtHelperService();

  startDate: string = '';
  endDate: string = '';
  createReservation(
    reservationId: any,
    accommodationId: any,
    fromDate: NgbDate,
    toDate: NgbDate
  ): Observable<any> {
    this.startDate = this.formatNgbDate(fromDate);
    console.log(this.startDate);
    this.endDate = this.formatNgbDate(toDate);
    return this.http.post(
      '/api/reservations/for_user',
      {
        reservationId: reservationId,
        accoId: accommodationId,
        startDate: this.startDate,
        endDate: this.endDate,
      },
      { headers: this.headers, responseType: 'json' }
    );
  } // kreiranje rezervacije za korisnika

  getReservationsByAccoId(id: any): Observable<any> {
    return this.http.get('/api/reservations/by_acco/' + id, {
      headers: this.headers,
      responseType: 'json'
    })
  }

  getReservations(id: any): Observable<any> {
    // dobalja sve rezervisane datume
    return this.http.get('/api/reservations/dates_by_acco_id/' + id, {
      headers: this.headers,
      responseType: 'json',
    });
  }

  createReservationDatesForAccomodation(
    id: any,
    reservation: any
  ): Observable<any> {
    let requestBody: any;

    if (reservation.priceType === 'night') {
      requestBody = {
        accoId: id,
        numberPeople: 2,
        priceByPeople: null, // Set to null when priceType is 'night'
        priceByAccommodation: reservation.price,
        startDate: reservation.availableFrom,
        endDate: reservation.availableUntil,
      };
    } else if (reservation.priceType === 'person') {
      requestBody = {
        accoId: id,
        numberPeople: 2,
        priceByPeople: reservation.price,
        priceByAccommodation: null, // Set to null when priceType is 'person'
        startDate: reservation.availableFrom,
        endDate: reservation.availableUntil,
      };
    } else {
      // Handle other cases if needed
    }

    return this.http.post('/api/reservations/for_acco', requestBody, {
      headers: this.headers,
      responseType: 'json',
    });
  }
  getAvailabelDatesForAccomodation(id: any): Observable<any> {
    // dobalja sve periode dostupnosti za akomodaciju
    return this.http.get(`${'/api/reservations/by_acco/'}${id}`, {
      headers: this.headers,
      responseType: 'json',
    });
  } // dobavljanje perioda dostupnosti

  formatNgbDate(date: NgbDate): string {
    if (date) {
      const formattedDate =
        date.year +
        '-' +
        this.padZero(date.month) +
        '-' +
        this.padZero(date.day);
      return formattedDate;
    }
    return '';
  }

  private padZero(value: number): string {
    return value < 10 ? '0' + value : value.toString();
  }

  getAllReservationsByUserId(): Observable<any> {
    return this.http.get(`/api/reservations/by_user`, {
      headers: this.headers,
      responseType: 'json',
    });
  } // dobijanje cele rezervacije po id usera

  getAllReservationDatesByDate(
    startDate: string,
    endDate: string
  ): Observable<any> {
    return this.http.get(
      '/api/reservations/search_by_date/' + startDate + '/' + endDate,
      {
        headers: this.headers,
        responseType: 'json',
      }
    );
  }

  cancelReservationsByUserId(reservation: any): Observable<any> {
    console.log(reservation);
    const locale = 'en-US';
    reservation.startDate = formatDate(
      new Date(reservation.startDate),
      'yyyy-MM-dd',
      locale
    );
    reservation.endDate = formatDate(
      new Date(reservation.endDate),
      'yyyy-MM-dd',
      locale
    );
    return this.http.patch(
      '/api/reservations/for_user',
      {
        reservationId: reservation.reservationId,
        accoId: reservation.accoId,
        price: reservation.price,
        startDate: reservation.startDate,
        endDate: reservation.endDate,
      },
      {}
    );
  }
}
