import { Component, Input, OnInit } from '@angular/core';
import { Accommodation } from '../../model/accommodation';
import { AccommodationService } from '../../service/accommodation.service';
import { AccoPicture } from '../../model/accoPicture';

@Component({
  selector: 'app-accommo-list',
  templateUrl: './accommo-list.component.html',
  styleUrls: ['./accommo-list.component.css'],
})
export class AccommoListComponent implements OnInit{
  @Input() accommodations: Accommodation[] = [];
  accommodationImages: any[string] = [];
  accommodationWithPictures: AccoPicture[] = [];
  price: number[] = []
  constructor (
    private accommodationService: AccommodationService
  ) {
    this.price = [12, 11, 9, 8, 18]
  }


  ngOnInit(): void {
    this.fillAccommodationWithPictures()
    this.getAccommodationImage()
  }

  sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  async fillAccommodationWithPictures(){
    await this.sleep(100);
    for (const accommodation of this.accommodations) {
      
      this.accommodationWithPictures.push(accommodation as AccoPicture)
    }
  }

  async getAccommodationImage(){
    await this.sleep(500)
    for (const accommodation2 of this.accommodationWithPictures) {
      console.log(accommodation2.images)
      this.accommodationService.getAccommodationImage(accommodation2.images[1]).subscribe(
        (blob: Blob) => {
          const reader = new FileReader();
          reader.onloadend = () => {
            const dataUrl = reader.result as string;
            this.accommodationImages.push(dataUrl);
            };
          reader.readAsDataURL(blob);
          },
          (error) => {
            console.error('Error fetching image:', error);
          });
      }  
    }
}