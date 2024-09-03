import { ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { ReservationService } from '../service/reservation.service';
import { AccommodationService } from '../service/accommodation.service';
import { ToastrService } from 'ngx-toastr';

@Component({
  selector: 'app-reservations',
  templateUrl: './reservations.component.html',
  styleUrls: ['./reservations.component.css'],
})
export class ReservationsComponent implements OnInit {
  userReservations: any[] = [];

  constructor(
    private router: Router,
    private accommodationService: AccommodationService,
    private reservationService: ReservationService,
    private toastr: ToastrService
  ) {}

  ngOnInit(): void {
    this.reservationService.getAllReservationsByUserId().subscribe({
      next: (data) => {
        this.userReservations = data;
      },
      error: (err) => {
        this.toastr.error('Cant get reservation for user');
        console.log(err.error.error);
        this.router.navigate(['']);
      },
    });
  }

  undoReservation(reservation: any) {
    this.reservationService.cancelReservationsByUserId(reservation).subscribe({
      next: (date) => {
        this.toastr.success('Successfully canceled reservation');
        this.ngOnInit(); // reload strane
      },
      error: (err) => {
        this.toastr.error(err.error.message);
      },
    });
  }
}
