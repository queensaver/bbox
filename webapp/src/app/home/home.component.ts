import { Component } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-home',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.scss']
})
export class HomeComponent {

  constructor(
    private http: HttpClient,
  ) {}

  loading: boolean = false;

  scan() {
    console.log('scan');
    this.loading = true;
    this.http.get('/scan').subscribe((res: any) => {
      console.log(res);
      this.loading = false;
    });
  }
}
