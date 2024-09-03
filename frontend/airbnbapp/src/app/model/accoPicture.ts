import { AmenityType } from './amenityType';
import { Location } from './location';

export interface AccoPicture {
  _id?: number;
  name?: string;
  location?: Location;
  minGuests?: number;
  maxGuests?: number;
  amenities?: AmenityType[];
  username?: string;
  AverageGrade?: string;
  images: string[];
}
