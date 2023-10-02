import { Component } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import {MatDialog, MAT_DIALOG_DATA, MatDialogRef, MatDialogModule} from '@angular/material/dialog';
import { ImageDialogComponent } from '../image-dialog/image-dialog.component';

@Component({
  selector: 'app-home',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.scss']
})
export class HomeComponent {

  constructor(
    private http: HttpClient,
    private dialog: MatDialog,
  ) {}

  loading: boolean = false;
  loaded: boolean = false;

  scan() {
    console.log('scan');
    this.loading = true;
    this.loaded = false;
    this.http.get('/scan').subscribe((res: any) => {
      console.log(res);
      this.loading = false;
      this.loaded = true;
    });
  }

  show_scan() {
    const dialogRef = this.dialog.open(ImageDialogComponent);
  }

}
