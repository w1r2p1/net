import { Component, OnInit, ViewEncapsulation, Input } from '@angular/core';
import { ImHistoryMessage } from '../../providers';

@Component({
  selector: 'app-im-history-view',
  templateUrl: './im-history-view.component.html',
  styleUrls: ['./im-history-view.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class ImHistoryViewComponent implements OnInit {
  list: Array<ImHistoryMessage>;
  constructor() { }
  ngOnInit() {

  }

}
