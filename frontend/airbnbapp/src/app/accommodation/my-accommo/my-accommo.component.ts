import { Component, OnInit } from '@angular/core';
import { Accommodation } from '../../model/accommodation';
import { AccommodationService } from '../../service/accommodation.service';
import { AuthService } from '../../service/auth.service';

@Component({
  selector: 'app-my-accommo',
  templateUrl: './my-accommo.component.html',
  styleUrls: ['./my-accommo.component.css'],
})
export class MyAccommoComponent implements OnInit {
  accommodations: Accommodation[] = [];
  constructor(
    private authService: AuthService,
    private accommodationService: AccommodationService
  ) {}

  ngOnInit(): void {
    let username = this.authService.getUsername();

    this.accommodationService.getAllByUsername(username).subscribe({
      next: (data) => {
        this.accommodations = data as Accommodation[];
        // console.log(this.accommodations);
      },
      error: (err) => {
        console.log(err);
      },
    });
  }
}
