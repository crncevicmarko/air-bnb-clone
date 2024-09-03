import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { Observable } from 'rxjs';
import { Location } from '../model/location';
import { Accommodation } from '../model/accommodation';

@Injectable({
  providedIn: 'root',
})
export class AccommodationService {
  constructor(private http: HttpClient, private router: Router) {}

  private headers = new HttpHeaders({ 'Content-Type': 'application/json' });
  private header = new HttpHeaders({ 'Content-Type': 'multipart/form-data' });

  getById(id: any): Observable<any> {
    return this.http.get('/api/accommodations/' + id, {
      headers: this.headers,
      responseType: 'json',
    });
  }

  getAll(): Observable<any> {
    return this.http.get('/api/accommodations/', {
      headers: this.headers,
      responseType: 'json',
    });
  }

  getAllRecommended(list: string[]): Observable<any> {
    return this.http.post(
      '/api/accommodations/recommendations',
      {
        list,
      },
      {
        headers: this.headers,
        responseType: 'json',
      }
    );
  }

  getAllByUsername(id: string): Observable<any> {
    return this.http.get('/api/accommodations/myAccommodations/' + id, {
      headers: this.headers,
      responseType: 'json',
    });
  }

  getAllAccommodationGrades(id: any): Observable<any> {
    return this.http.get(`${'/api/accommodations/accommodationGrades/'}${id}`, {
      headers: this.headers,
      responseType: 'json',
    });
  }

  deleteAccommodationGrade(id: any): Observable<any> {
    return this.http.delete(
      `${'/api/accommodations/deleteAccommodationGrade/'}${id}`,
      {
        headers: this.headers,
        responseType: 'json',
      }
    );
  }

  getAllByLocation(location: string): Observable<any> {
    return this.http.get('/api/accommodations/search_by_location/' + location, {
      headers: this.headers,
      responseType: 'json',
    });
  }

  getAllByNoGuests(noGuests: string): Observable<any> {
    return this.http.get('/api/accommodations/search_by_noGuests/' + noGuests, {
      headers: this.headers,
      responseType: 'json',
    });
  }

  getAllByDate(startDate: string, endDate: string): Observable<any> {
    return this.http.get(
      '/api/accommodations/search_by_date/' + startDate + '/' + endDate,
      {
        headers: this.header,
        responseType: 'json',
      }
    );
  }

  gradeAccommodation(id: any, grade: any): Observable<any> {
    console.log(id);
    return this.http.post(
      `/api/accommodations/accommodationGrade`,
      {
        accommodationId: id,
        grade: grade,
      },
      {
        headers: this.headers,
        responseType: 'json',
      }
    );
  }

  getAccommodationImage(image: string): Observable<any> {
    return this.http.get('/api/accommodations/read/' + image, {
      headers: this.header,
      responseType: 'blob',
    });
  }

  // getAllByNoGuests(noGuests: string): Observable<any> {
  //   return this.http.get('/api/accommodations/search_by_noGuests/' + noGuests, {
  //     headers: this.headers,
  //     responseType: 'json',
  //   });
  // }

  insert(uname: string, accommodation: any, imageNames: any): Observable<any> {
    let requestBody: any;

    if (accommodation.priceType === 'night') {
      requestBody = {
        name: accommodation.name,
        location: {
          country: accommodation.location!.country,
          city: accommodation.location!.city,
          streetName: accommodation.location!.streetName,
          streetNumber: accommodation.location!.streetNumber,
        },
        amenities: accommodation.amenities,
        minGuests: accommodation.minGuests,
        maxGuests: accommodation.maxGuests,
        username: uname,
        images: imageNames,
        numberPeople: 2,
        priceByPeople: null,
        priceByAccommodation: accommodation.price,
        startDate: accommodation.availableFrom,
        endDate: accommodation.availableUntil,
      };
    } else if (accommodation.priceType === 'person') {
      requestBody = {
        name: accommodation.name,
        location: {
          country: accommodation.location!.country,
          city: accommodation.location!.city,
          streetName: accommodation.location!.streetName,
          streetNumber: accommodation.location!.streetNumber,
        },
        amenities: accommodation.amenities,
        minGuests: accommodation.minGuests,
        maxGuests: accommodation.maxGuests,
        username: uname,
        images: imageNames,
        numberPeople: 2,
        priceByPeople: accommodation.price,
        priceByAccommodation: null,
        startDate: accommodation.availableFrom,
        endDate: accommodation.availableUntil,
      };
    }
    return this.http.post('/api/accommodations/create', requestBody, {
      headers: this.headers,
      responseType: 'json',
    });
  }

  createImages(images: any): Observable<any> {
    const formData: FormData = new FormData();

    images.forEach((file: File) => {
      formData.append('files', file, file.name);
    });

    return this.http.post('/api/accommodations/write', formData, {
      headers: this.header,
      responseType: 'json',
    });
  }
}
