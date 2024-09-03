import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AccommodationService } from '../../service/accommodation.service';
import { Accommodation } from '../../model/accommodation';
import { NgbDate, NgbDateStruct } from '@ng-bootstrap/ng-bootstrap';
import { AuthService } from '../../service/auth.service';
import { ReservationService } from '../../service/reservation.service';
import { NormalDateRange } from '../../model/normalDateRange';
import { NgbDateRange } from '../../model/NgbDateRange';
import {
  FormBuilder,
  FormGroup,
  ValidationErrors,
  Validators,
} from '@angular/forms';
import { ProfServiceService } from '../../service/prof.service.service';
import { NotificationService } from '../../service/notification.service';
import { Notification1 } from '../../model/notification';
import { ToastrService } from 'ngx-toastr';
import { RecommendationService } from 'src/app/service/recommendation.service';

@Component({
  selector: 'app-accommo-info',
  templateUrl: './accommo-info.component.html',
  styleUrls: ['./accommo-info.component.css'],
})
export class AccommoInfoComponent implements OnInit {
  grades: any[] = [];
  accommodationGrades: any[] = [];
  form: FormGroup;
  accommodationForm!: FormGroup;
  formAccommodation!: FormGroup;
  hostAverageGrade: any;
  constructor(
    private fb: FormBuilder,
    private route: ActivatedRoute,
    private router: Router,
    private profService: ProfServiceService,
    private accommodationService: AccommodationService,
    private authService: AuthService,
    private reservationService: ReservationService,
    private notificationService: NotificationService,
    private toastr: ToastrService,
    private recommendationService: RecommendationService
  ) {
    this.form = this.fb.group({
      grade: [
        null,
        [Validators.required, Validators.min(1), Validators.max(5)],
      ],
    });
    this.formAccommodation = this.fb.group({
      grade: [
        null,
        [Validators.required, Validators.min(1), Validators.max(5)],
      ],
    });
  }

  role: string = '';
  username: string = '';

  id: number = 0;
  reservationId: string = '';
  accommodation: Accommodation = {};
  hostId: string = '';
  accommodationImages: any[string] = [];

  isDataEmpty = false;

  notification: Notification1 = {
    hostId: '',
    description: '',
  };

  dateList: NormalDateRange[] = [];

  fromDisDate: NgbDate | null = null;
  toDisDate: NgbDate | null = null;
  startDate: NgbDate | null = null;
  endDate: NgbDate | null = null;
  hoveredDate: NgbDate | null = null;
  fromDate: NgbDate | null = null;
  toDate: NgbDate | null = null;

  //Lista dostupnih termina
  allDateAvailabilityList: NgbDateRange[] = [];

  //lista zakazanih termina
  disabledDateList: NgbDateRange[] = [];

  //prvi i poslednji dostupni datumi
  firstAvailableDate: NgbDate | null = null;
  lastAvailableDate: NgbDate | null = null;

  ngOnInit(): void {
    this.accommodationForm = this.fb.group({
      availableFrom: ['', Validators.required],
      availableUntil: ['', Validators.required],
      priceType: ['', Validators.required], // Make sure it's bound to priceType
      price: ['', Validators.required], // Make sure it's bound to price
    });

    this.route.params.subscribe((params) => {
      this.id = params['id'];
    });

    this.role = this.authService.getRole();
    this.username = this.authService.getUsername();

    this.accommodationService.getById(this.id).subscribe({
      next: (data) => {
        this.accommodation = data;
        console.log('Accommodation:', this.accommodation);
        for (const image of data.images) {
          // console.log(image);
          this.accommodationService.getAccommodationImage(image).subscribe(
            (blob: Blob) => {
              // console.log('Blob:', blob);
              const reader = new FileReader();
              reader.onloadend = () => {
                const dataUrl = reader.result as string;
                // Now 'dataUrl' contains the data URL representation of the image
                // You can use 'dataUrl' as needed in your application
                this.accommodationImages.push(dataUrl);
                // console.log(dataUrl);
              };
              reader.readAsDataURL(blob);
            },
            (error) => {
              console.error('Error fetching image:', error);
            }
          );
        }

        // this.authService.getUserById()
        // console.log(this.accommodation._id);
        // console.log(data);
      },
      error: (err) => {
        this.toastr.error(err.error.message);
        this.isDataEmpty = true;
        this.router.navigate(['']);
      },
    });

    this.reservationService
      .getAvailabelDatesForAccomodation(this.id)
      .subscribe({
        next: (data) => {
          console.log('Available dates: ', data);
          this.reservationId = data[0].reservationId;
          this.hostId = data[0].userId;
          // console.log('HostId1:', this.hostId);
          this.profService.getAllHostGrades(this.hostId).subscribe({
            next: (data) => {
              // console.log(data);
              this.grades = data;
            },
            error: (err) => {
              this.toastr.error(err.error.message);
            },
          });
          this.accommodationService
            .getAllAccommodationGrades(this.id)
            .subscribe({
              next: (data) => {
                // console.log(data);
                this.accommodationGrades = data;
              },
              error: (err) => {
                this.toastr.error(err.error.message);
              },
            });
          this.authService.getUserById(this.hostId).subscribe({
            next: (data) => {
              // console.log('host:', data);
              this.hostAverageGrade = data.averageGrade;
            },
            error: (err) => {
              this.toastr.error(err.error.message);
            },
          });
          //pretvori sve termine iz baze u ngbDate
          for (let availableDatePeriod of data) {
            let startDate = new NgbDate(
              new Date(availableDatePeriod.startDate).getFullYear(),
              new Date(availableDatePeriod.startDate).getUTCMonth() + 1,
              new Date(availableDatePeriod.startDate).getUTCDate()
            );

            this.startDate = startDate;

            let endDate = new NgbDate(
              new Date(availableDatePeriod.endDate).getFullYear(),
              new Date(availableDatePeriod.endDate).getUTCMonth() + 1,
              new Date(availableDatePeriod.endDate).getUTCDate()
            );

            this.endDate = endDate;

            //jedan termin dostupnosti
            let newAvailablePeriod: NgbDateRange = { startDate, endDate };

            this.allDateAvailabilityList.push(newAvailablePeriod);
          }

          if (this.allDateAvailabilityList.length != 0) {
            this.firstAvailableDate =
              this.allDateAvailabilityList[0].startDate!;

            this.lastAvailableDate =
              this.allDateAvailabilityList[
                this.allDateAvailabilityList.length - 1
              ].endDate!;
          }
        },
        error: (err) => {
          console.log(err);
          this.toastr.error(err.error.message);
          // this.router.navigate(['']);
        },
      });
    this.reservationService.getReservations(this.id).subscribe({
      next: (data) => {
        // console.log(data);
        if (data != null) {
          this.dateList = data as NormalDateRange[];
          for (let dateRange of this.dateList) {
            let startDate = new NgbDate(
              new Date(dateRange.begin_accomodation_date!).getFullYear(),
              new Date(dateRange.begin_accomodation_date!).getUTCMonth() + 1,
              new Date(dateRange.begin_accomodation_date!).getUTCDate()
            );
            let endDate = new NgbDate(
              new Date(dateRange.end_accomodation_date!).getFullYear(),
              new Date(dateRange.end_accomodation_date!).getUTCMonth() + 1,
              new Date(dateRange.end_accomodation_date!).getUTCDate()
            );
            let disabledDateRange: NgbDateRange = { startDate, endDate };
            this.disabledDateList.push(disabledDateRange);
          }
          console.log(this.disabledDateList);
        }
      },
      error: (err) => {
        console.log(err);
        this.toastr.error(err.error.messa);
      },
    });
  }

  isDisabled = (date: NgbDate, current?: { month: number }) => {
    //provera za rezervisane datume
    for (let dateRange of this.disabledDateList) {
      if (
        (date.after(dateRange.startDate) && date.before(dateRange.endDate)) ||
        date.equals(dateRange.startDate) ||
        date.equals(dateRange.endDate)
      )
        return true;
    }

    let counter = 1;

    //provera za slobodne datume
    for (let dateRange of this.allDateAvailabilityList) {
      if (
        (date.after(dateRange.startDate) && date.before(dateRange.endDate)) ||
        date.equals(dateRange.startDate) ||
        date.equals(dateRange.endDate)
      ) {
        break;
      } else {
        if (counter < this.allDateAvailabilityList.length) {
          counter++;
          continue;
        } else return true;
      }
    }
    return (
      date.before(this.firstAvailableDate) && date.after(this.lastAvailableDate)
    );
  };

  onDateSelection(date: NgbDate) {
    if (this.disabledDateList.length > 0) {
      for (let blackDateRange of this.disabledDateList) {
        if (!this.fromDate && !this.toDate) {
          this.fromDate = date;
        } else if (
          this.fromDate &&
          !this.toDate &&
          date.after(this.fromDate) &&
          ((this.fromDate.before(blackDateRange.startDate) &&
            date.before(blackDateRange.startDate)) ||
            (this.fromDate.after(blackDateRange.endDate) &&
              date.after(blackDateRange.endDate)))
        ) {
          this.toDate = date;
        } else if (
          this.fromDate &&
          this.toDate &&
          ((this.fromDate.before(blackDateRange.startDate) &&
            date.before(blackDateRange.startDate)) ||
            (this.fromDate.after(blackDateRange.endDate) &&
              date.after(blackDateRange.endDate)))
        ) {
          continue;
        } else {
          this.toDate = null;
          this.fromDate = date;
        }
      }
    } else {
      if (!this.fromDate && !this.toDate) {
        this.fromDate = date;
      } else if (this.fromDate && !this.toDate && date.after(this.fromDate)) {
        this.toDate = date;
      } else {
        this.toDate = null;
        this.fromDate = date;
      }
    }
  }

  isHovered(date: NgbDate) {
    return (
      this.fromDate &&
      !this.toDate &&
      this.hoveredDate &&
      date.after(this.fromDate) &&
      date.before(this.hoveredDate)
    );
  }

  isInside(date: NgbDate) {
    return this.toDate && date.after(this.fromDate) && date.before(this.toDate);
  }

  isRange(date: NgbDate) {
    return (
      date.equals(this.fromDate) ||
      (this.toDate && date.equals(this.toDate)) ||
      this.isInside(date) ||
      this.isHovered(date)
    );
  }

  submitGrade() {
    this.profService.gradeHost(this.hostId, this.form.value.grade).subscribe({
      next: (data) => {
        this.toastr.success('Host graded');
        this.form.reset();
        this.ngOnInit();
      },
      error: (err) => {
        this.toastr.error(err.error.message);
        this.form.reset();
      },
    });

    this.createNotification('One of your guests gave a review on you!');
  }

  reserveDates() {
    this.reservationService
      .createReservation(
        this.reservationId,
        this.accommodation._id!,
        this.fromDate!,
        this.toDate!
      )
      .subscribe({
        next: (data) => {
          this.toastr.success('Successfully reserved accommodation');
          this.recommendationService.insert(this.accommodation).subscribe({
            next: (data) => {
              console.log('Sent to Recommendation service');
            },
            error: (err) => {
              console.log(err);
            },
          });
          alert('Reserved');
          this.router.navigate(['']);
        },
        error: (err) => {
          this.toastr.error(err.error.message);
        },
      });

    this.createNotification('One of your accommodations just got reserved!');
  }
  deleteHostGrade(id: any) {
    this.profService.deleteHostGrades(id).subscribe({
      next: (data) => {
        this.toastr.success('Deleted host review');
        this.ngOnInit();
      },
      error: (err) => {
        this.toastr.error(err.error.message);
      },
    });

    this.createNotification('One of your guests deleted their review on you!');
  }
  deleteAccommodationGrade(id: any) {
    this.accommodationService.deleteAccommodationGrade(id).subscribe({
      next: (data) => {
        this.toastr.success('Deleted accommodation review');
        this.ngOnInit();
      },
      error: (err) => {
        this.toastr.error(err.error.message);
      },
    });
    this.createNotification(
      'One of your guests deleted their review on one of your accommodations!'
    );
  }

  submitAccommodationGrade() {
    this.accommodationService
      .gradeAccommodation(this.id, this.formAccommodation.value.grade)
      .subscribe({
        next: (data) => {
          this.toastr.success('Successfully created accommodation review');
          this.formAccommodation.reset();
          this.ngOnInit();
        },
        error: (err) => {
          this.formAccommodation.reset();
          this.toastr.error(err.error.message);
        },
      });

    this.createNotification(
      'One of your guests left a review on one of your accommodations!'
    );
  }

  createNotification(description: string) {
    this.notification.hostId = this.hostId;
    this.notification.description = description;
    this.notificationService.createNotification(this.notification).subscribe({
      next: (data) => {
        console.log('Notification Sent');
      },
      error: (err) => {
        this.toastr.warning(err.error.message);
      },
    });
  }

  onSubmit() {
    console.log(this.accommodation._id, this.accommodationForm.value);
    this.reservationService
      .createReservationDatesForAccomodation(
        this.accommodation._id,
        this.accommodationForm.value
      )
      .subscribe({
        next: (data) => {
          this.toastr.success('Reservation is successfully created.');
          this.router.navigate(['accommodations/myAccommodations']);
        },
        error: (err) => {
          this.toastr.error(err.error);
        },
      });
  }
}
