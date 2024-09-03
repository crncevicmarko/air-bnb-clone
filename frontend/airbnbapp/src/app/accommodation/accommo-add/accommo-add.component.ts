import { Component, OnInit } from '@angular/core';
import {
  AbstractControl,
  FormArray,
  FormBuilder,
  FormControl,
  FormGroup,
  ValidatorFn,
  Validators,
} from '@angular/forms';
import { Router } from '@angular/router';
import { AccommodationService } from '../../service/accommodation.service';
import { Accommodation } from '../../model/accommodation';
import { AmenityType } from '../../model/amenityType';
import { AuthService } from '../../service/auth.service';
import { ToastrService } from 'ngx-toastr';

@Component({
  selector: 'app-accommo-add',
  templateUrl: './accommo-add.component.html',
  styleUrls: ['./accommo-add.component.css'],
})
export class AccommoAddComponent implements OnInit {
  form!: FormGroup;
  amenityRange = AmenityType;

  imageUrls: Array<string | null> = new Array(5).fill(null);
  imageNames: Array<string> = [];
  selectedFiles: File[] = [];
  errorMessage: string = '';
  imageCount: number = 0;
  // accommodationForm!: FormGroup;

  onFileChange(event: any) {
    this.errorMessage = '';
    this.imageUrls = new Array(5).fill(null);
    const files = event.target.files;
    console.log('fileLength:', files.length);

    if (files && files.length === 5) {
      this.selectedFiles = Array.from(files);
      this.imageCount = 5;

      const imagesArray = this.form.get('images') as FormArray;
      imagesArray.clear();

      for (let i = 0; i < 5; i++) {
        const file = files[i];
        const reader = new FileReader();

        reader.onload = (e) => {
          if (e.target?.result) {
            this.imageUrls[i] = e.target.result as string;
            this.imageNames[i] = file.name;
            imagesArray.push(new FormControl(file)); // Add the image to the FormArray
          }
        };
        reader.readAsDataURL(file);
      }
    } else {
      this.imageCount = files ? files.length : 0;
      this.errorMessage = 'Please choose exactly 5 images.';
      console.log('Please choose exactly 5 images.');
    }
  }

  constructor(
    private fb: FormBuilder,
    private router: Router,
    private accommodationService: AccommodationService,
    private authService: AuthService,
    private toastr: ToastrService
  ) {}

  ngOnInit(): void {
    this.form = this.fb.group(
      {
        name: [null, Validators.required],
        location: this.fb.group({
          country: [null, Validators.required],
          city: [null, Validators.required],
          streetName: [null, Validators.required],
          streetNumber: [null, Validators.required],
        }),
        minGuests: [null, Validators.required],
        maxGuests: [null, Validators.required],
        amenities: new FormArray([], Validators.required),
        images: new FormArray([], [Validators.required]),
        availableFrom: ['', Validators.required],
        availableUntil: ['', Validators.required],
        priceType: ['', Validators.required],
        price: ['', Validators.required],
      },
      { validators: [this.imageCountValidator.bind(this)] }
    );
    // }

    // Subscribe to changes in the 'images' FormArray
    this.form.get('images')?.valueChanges.subscribe(() => {
      this.form.get('images')?.updateValueAndValidity();
    });
  }

  // minSelectedImages(minImages: number): ValidatorFn {
  //   return (control: AbstractControl): { [key: string]: any } | null => {
  //     const selectedImages = control.value.filter(Boolean).length;

  //     return selectedImages >= minImages ? null : { minImages: true };
  //   };
  // }

  imageCountValidator(control: AbstractControl): { [key: string]: any } | null {
    const formArray = control.get('images') as FormArray;
    const count = formArray ? formArray.length : 0;
    return count === 5 ? null : { invalidImageCount: true };
  }

  submit() {
    // let accommodation: Accommodation = { ...this.form.value };
    // accommodation.username = this.authService.getUsername();
    console.log('ImageNames:', this.imageNames);
    console.log('Images:', this.selectedFiles);
    console.log('Accommodation:', this.form.value);

    this.accommodationService
      .insert(this.authService.getUsername(), this.form.value, this.imageNames)
      .subscribe({
        next: (data) => {
          console.log('create success');
          this.router.navigate(['']);
        },
        error: (err) => {
          alert(err.error.message);
          console.log(err);
        },
      });

    this.accommodationService.createImages(this.selectedFiles).subscribe({
      next: (data) => {
        this.router.navigate(['']);
      },
      error: (err) => {
        console.log(err);
        alert(err.error.message);
      },
    });
  }

  getRange(obj: any) {
    return Object.values(obj);
  }

  onCheckChange(event: any) {
    const formArray: FormArray = this.form.get('amenities') as FormArray;

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
