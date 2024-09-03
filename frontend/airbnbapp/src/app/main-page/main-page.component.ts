import { Component, OnDestroy, OnInit } from '@angular/core';
import { Accommodation } from '../model/accommodation';
import { AccommodationService } from '../service/accommodation.service';
import { Router } from '@angular/router';
import { AuthService } from '../service/auth.service';
import { Subscription, catchError, forkJoin, of, range } from 'rxjs';
import {
  AbstractControl,
  FormArray,
  FormControl,
  FormGroup,
} from '@angular/forms';
import { NgbDate } from '@ng-bootstrap/ng-bootstrap';
import { ReservationService } from '../service/reservation.service';
import { ReservationByDateSearch } from '../model/reservationByDateSearch';
import { AmenityType } from '../model/amenityType';
import { ToastRef, ToastrService } from 'ngx-toastr';
import { RecommendationService } from '../service/recommendation.service';

@Component({
  selector: 'app-main-page',
  templateUrl: './main-page.component.html',
  styleUrls: ['./main-page.component.css'],
})
export class MainPageComponent implements OnInit, OnDestroy {
  accommodations: Accommodation[] = [];
  recommendations: string[] = [];
  recommendedAccomo: Accommodation[] = [];
  recCount: number = 0;

  isLoggedin: boolean = false;
  userRole: string = '';
  logSub: Subscription;
  rolesub: Subscription;

  //Search
  searchAccoForm: FormGroup;
  filterAccoForm: FormGroup;
  locationInput: string = '';
  noGuestsInput: string = '';
  startDateInput: string = '';
  endDateInput: string = '';
  accommodationsByLocation: Accommodation[] = [];
  accommodationsByNoGuest: Accommodation[] = [];
  accommodationsByDate: Accommodation[] = [];

  //Filter
  amenityRange = AmenityType;
  priceFrom: number = 0;
  priceTo: number = 0;
  isFeatured: boolean = false;
  amenities: [] = [];
  userId: number = 0;
  soonToBeAccommodations: Accommodation[] = [];
  priceAccommodations: Accommodation[] = [];
  featuredHostAccommodations: Accommodation[] = [];
  amenitiesAccommodations: Accommodation[] = [];
  isFilterPrice: boolean = false;
  isFilterAmenities: boolean = false;
  isFilterFeaturedHost: boolean = false;

  constructor(
    private router: Router,
    private accommodationService: AccommodationService,
    private authService: AuthService,
    private reservationService: ReservationService,
    private toastr: ToastrService,
    private recommendationService: RecommendationService
  ) {
    this.logSub = this.authService.isLoggedin.subscribe(
      (data) => (this.isLoggedin = data)
    );
    this.rolesub = this.authService.role.subscribe(
      (data) => (this.userRole = data)
    );

    this.searchAccoForm = new FormGroup({
      location: new FormControl(),
      startDate: new FormControl(),
      endDate: new FormControl(),
      noGuests: new FormControl(),
    });

    this.filterAccoForm = new FormGroup({
      priceFrom: new FormControl(),
      priceTo: new FormControl(),
      amenities: new FormArray([]),
      isFeatured: new FormControl(),
    });
  }

  ngOnInit(): void {
    this.authService.checkLoggin();
    this.authService.checkRole();

    this.accommodationService.getAll().subscribe({
      next: (data) => {
        this.accommodations = data as Accommodation[];
        if (this.userRole == 'GUEST')
          this.recommendationService.getAllRecommendations().subscribe({
            next: (data) => {
              // console.log(data);
              this.accommodationService.getAllRecommended(data).subscribe({
                next: (data) => {
                  // console.log(data);
                  if (data != null) {
                    this.recommendedAccomo = data;
                    this.recCount = data.length;
                  }
                },
                error: (err) => {
                  console.log(err);
                },
              });
            },
            error: (err) => {
              console.log(err);
            },
          });
        // console.log(this.accommodations);
      },
      error: (err) => {
        this.toastr.error(err);
      },
    });
  }

  ngOnDestroy(): void {
    this.logSub.unsubscribe();
    this.rolesub.unsubscribe();
  }

  filterAcco(): void {
    this.isFilterAmenities = false;
    this.isFilterFeaturedHost = false;
    this.isFilterPrice = false;
    this.soonToBeAccommodations = [];
    this.featuredHostAccommodations = [];
    this.amenitiesAccommodations = [];
    this.priceAccommodations = [];

    this.priceFrom = this.filterAccoForm.get('priceFrom')?.value;
    this.priceTo = this.filterAccoForm.get('priceTo')?.value;
    this.amenities = this.filterAccoForm.get('amenities')?.value;
    this.isFeatured = this.filterAccoForm.get('isFeatured')?.value;

    const observables = [];

    if (this.priceFrom > 0 && this.priceTo > 0) {
      this.isFilterPrice = true;
      for (const accommodation of this.accommodations) {
        this.reservationService
          .getReservationsByAccoId(accommodation._id)
          .subscribe({
            next: (data) => {
              for (const reservation of data) {
                if (
                  (reservation.priceByAccommodation >= this.priceFrom ||
                    reservation.priceByPeople >= this.priceFrom) &&
                  (reservation.priceByAccommodation <= this.priceTo ||
                    reservation.priceByPeople <= this.priceTo)
                ) {
                  this.priceAccommodations.push(accommodation);
                }
              }
            },
          });
      }
      if (this.priceAccommodations.length <= 0) {
        this.toastr.info(
          'No accommodations in that price range have been found'
        );
      }
      this.priceFrom = 0;
      this.priceTo = 0;
    }

    if (this.amenities.length > 0) {
      this.isFilterAmenities = true;
      for (const accommodation of this.accommodations) {
        if (accommodation.amenities) {
          for (const amenity of this.amenities) {
            if (accommodation.amenities.includes(amenity)) {
              if (!this.amenitiesAccommodations.includes(accommodation)) {
                this.amenitiesAccommodations.push(accommodation);
              }
            }
          }
        }
      }
      if (this.amenitiesAccommodations.length <= 0) {
        this.toastr.info(
          'No accommodations with those amenities have been found'
        );
      }
      this.amenities = [];
    }

    if (this.isFeatured == true) {
      this.isFilterFeaturedHost = true;
      for (const accommodation of this.accommodations) {
        this.authService
          .getUserByUsername(accommodation.username ?? '')
          .subscribe({
            next: (data) => {
              this.authService.getUserById(data._id).subscribe({
                next: (data) => {
                  if (data.isFeatured == true) {
                    this.featuredHostAccommodations.push(accommodation);
                  }
                },
              });
            },
          });
      }
      if (this.featuredHostAccommodations.length <= 0) {
        this.toastr.info(
          'No accommodations with featured hosts have been found'
        );
      }
    }
    this.displayFilteredAccos();
  }

  sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  async displayFilteredAccos() {
    await this.sleep(500);
    if (
      this.isFilterPrice == true &&
      this.isFilterAmenities == false &&
      this.isFilterFeaturedHost == false
    ) {
      console.log('Price Filter');
      this.accommodations = this.priceAccommodations;
    } else if (
      this.isFilterPrice == true &&
      this.isFilterAmenities == true &&
      this.isFilterFeaturedHost == true
    ) {
      console.log('Everything Filter');
      for (const accommodation1 of this.priceAccommodations) {
        for (const accommodation2 of this.featuredHostAccommodations) {
          for (const accommodation3 of this.amenitiesAccommodations) {
            if (
              accommodation1._id == accommodation2._id &&
              accommodation1._id == accommodation3._id
            ) {
              this.soonToBeAccommodations.push(accommodation1);
            }
          }
        }
      }
      this.accommodations = this.soonToBeAccommodations;
    } else if (
      this.isFilterPrice == false &&
      this.isFilterAmenities == true &&
      this.isFilterFeaturedHost == false
    ) {
      console.log('Amenities Filter');
      this.accommodations = this.amenitiesAccommodations;
    } else if (
      this.isFilterPrice == false &&
      this.isFilterAmenities == false &&
      this.isFilterFeaturedHost == true
    ) {
      console.log('Featured Host Filter');
      this.accommodations = this.featuredHostAccommodations;
    } else if (
      this.isFilterPrice == true &&
      this.isFilterAmenities == true &&
      this.isFilterFeaturedHost == false
    ) {
      console.log('Price and Amenities');
      for (const accommodation1 of this.amenitiesAccommodations) {
        for (const accommodation2 of this.priceAccommodations) {
          if (accommodation1._id == accommodation2._id) {
            this.soonToBeAccommodations.push(accommodation1);
          }
        }
      }
      this.accommodations = this.soonToBeAccommodations;
    } else if (
      this.isFilterPrice == true &&
      this.isFilterAmenities == false &&
      this.isFilterFeaturedHost == true
    ) {
      console.log('Price and Featured Host Filter');
      for (const accommodation1 of this.priceAccommodations) {
        for (const accommodation2 of this.featuredHostAccommodations) {
          if (accommodation1._id == accommodation2._id) {
            this.soonToBeAccommodations.push(accommodation1);
          }
        }
      }
      this.accommodations = this.soonToBeAccommodations;
    } else if (
      this.isFilterPrice == false &&
      this.isFilterAmenities == true &&
      this.isFilterFeaturedHost == true
    ) {
      console.log('Featured Host and Amenities Filter');
      for (const accommodation1 of this.featuredHostAccommodations) {
        for (const accommodation2 of this.amenitiesAccommodations) {
          if (accommodation1._id == accommodation2._id) {
            this.soonToBeAccommodations.push(accommodation1);
          }
        }
      }
      this.accommodations = this.soonToBeAccommodations;
    }

    // this.isFilterAmenities = false;
    // this.isFilterFeaturedHost = false;
    // this.isFilterPrice = false;
    // this.soonToBeAccommodations = [];
    // this.featuredHostAccommodations = [];
    // this.amenitiesAccommodations = [];
    // this.priceAccommodations = [];
  }

  searchAcco(): void {
    this.locationInput = this.searchAccoForm.get('location')?.value;
    this.startDateInput = this.searchAccoForm.get('startDate')?.value;
    this.endDateInput = this.searchAccoForm.get('endDate')?.value;
    this.noGuestsInput = this.searchAccoForm.get('noGuests')?.value;

    const locationObservable =
      this.locationInput != null && this.locationInput !== ''
        ? this.accommodationService.getAllByLocation(this.locationInput)
        : of([]);

    const noGuestsObservable =
      this.noGuestsInput != null && this.noGuestsInput !== ''
        ? this.accommodationService.getAllByNoGuests(this.noGuestsInput)
        : of([]);

    const dateObservable =
      this.startDateInput != null && this.endDateInput != null
        ? this.accommodationService.getAllByDate(
            this.startDateInput,
            this.endDateInput
          )
        : of([]);

    forkJoin([
      locationObservable,
      noGuestsObservable,
      dateObservable,
    ]).subscribe({
      next: ([locations, noGuests, accoDate]: [
        Accommodation[],
        Accommodation[],
        Accommodation[]
      ]) => {
        this.accommodationsByLocation = locations as Accommodation[];
        this.accommodationsByNoGuest = noGuests as Accommodation[];
        this.accommodationsByDate = accoDate as Accommodation[];

        if (this.accommodationsByLocation == null) {
          this.toastr.info(
            'No accommodations with that location have been found'
          );
          this.accommodations = [];
        }

        if (this.accommodationsByDate == null) {
          this.toastr.info(
            'No accommodations with those reservation dates have been found'
          );
          this.accommodations = [];
        }

        if (this.accommodationsByNoGuest == null) {
          this.toastr.info(
            'No accommodations with that number of guests have been found'
          );
          this.accommodations = [];
        }

        if (
          this.accommodationsByLocation.length > 0 &&
          this.accommodationsByNoGuest.length == 0 &&
          this.accommodationsByDate.length == 0
        ) {
          console.log('Search by location only');
          this.accommodations = this.accommodationsByLocation;
        } else if (
          this.accommodationsByLocation.length > 0 &&
          this.accommodationsByDate.length > 0 &&
          this.accommodationsByNoGuest.length > 0
        ) {
          console.log('Search by all three');
          const tempList: Accommodation[] = [];
          for (const accoNoGuest of this.accommodationsByNoGuest) {
            for (const accoDate of this.accommodationsByDate) {
              for (const accoLocation of this.accommodationsByLocation) {
                if (
                  accoLocation._id == accoNoGuest._id &&
                  accoLocation._id == accoDate._id
                ) {
                  tempList.push(accoLocation);
                } else {
                  continue;
                }
              }
            }
          }
          this.accommodations = tempList;
        } else if (
          this.accommodationsByDate.length > 0 &&
          this.accommodationsByNoGuest.length == 0 &&
          this.accommodationsByLocation.length == 0
        ) {
          console.log('Search by date only');
          this.accommodations = this.accommodationsByDate;
        } else if (
          this.accommodationsByLocation.length == 0 &&
          this.accommodationsByNoGuest.length > 0 &&
          this.accommodationsByDate.length == 0
        ) {
          console.log('Search by number of guests only');
          this.accommodations = this.accommodationsByNoGuest;
        } else if (
          this.accommodationsByLocation.length > 0 &&
          this.accommodationsByNoGuest.length > 0
        ) {
          console.log('Search by location and number of guests');
          const tempList: Accommodation[] = [];
          for (const accoLocation of this.accommodationsByLocation) {
            for (const accoNoGuest of this.accommodationsByNoGuest) {
              if (accoLocation._id == accoNoGuest._id) {
                tempList.push(accoLocation);
              } else {
                continue;
              }
            }
          }
          this.accommodations = tempList;
        } else if (
          this.accommodationsByLocation.length > 0 &&
          this.accommodationsByDate.length > 0 &&
          this.accommodationsByNoGuest.length == 0
        ) {
          console.log('Search by location and date');
          const tempList: Accommodation[] = [];
          for (const accoLocation of this.accommodationsByLocation) {
            for (const accoNoGuest of this.accommodationsByDate) {
              if (accoLocation._id == accoNoGuest._id) {
                tempList.push(accoLocation);
              } else {
                continue;
              }
            }
          }
          this.accommodations = tempList;
        } else if (
          this.accommodationsByLocation.length == 0 &&
          this.accommodationsByDate.length > 0 &&
          this.accommodationsByNoGuest.length > 0
        ) {
          console.log('Search by date and number of guests');
          const tempList: Accommodation[] = [];
          for (const accoLocation of this.accommodationsByNoGuest) {
            for (const accoNoGuest of this.accommodationsByDate) {
              if (accoLocation._id == accoNoGuest._id) {
                tempList.push(accoLocation);
              } else {
                continue;
              }
            }
          }
          this.accommodations = tempList;
        } else if (
          this.accommodationsByLocation.length == 0 &&
          this.accommodationsByNoGuest.length == 0 &&
          this.accommodationsByDate.length == 0
        ) {
          this.ngOnInit();
        }

        this.accommodationsByLocation = [];
        this.accommodationsByNoGuest = [];
        this.accommodationsByDate = [];
      },
      error: (err) => {
        console.log(err);
      },
      complete: () => {
        console.log('Both observables complete');
      },
    });
  }

  getRange(obj: any) {
    return Object.values(obj);
  }

  onCheckChange(event: any) {
    const formArray: FormArray = this.filterAccoForm.get(
      'amenities'
    ) as FormArray;

    if (event.target.checked) {
      formArray.push(new FormControl(event.target.value));
    } else {
      let i: number = 0;

      formArray.controls.forEach(
        (ctrl: AbstractControl<any>, index: number) => {
          if (ctrl.value == event.target.value) {
            formArray.removeAt(index);
            return;
          }
        }
      );
    }
  }
}
