import { Component, OnInit, Input, ViewEncapsulation, HostListener, HostBinding, Output, EventEmitter } from '@angular/core';

@Component({
  selector: 'app-im-recent-item',
  templateUrl: './im-recent-item.component.html',
  styleUrls: ['./im-recent-item.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class ImRecentItemComponent implements OnInit {
  @Input() name = 'General User';
  @HostBinding('class.item-active') active = false;
  @Output('onClick') onClick: EventEmitter<ImRecentItemComponent> = new EventEmitter();
  constructor() { }

  ngOnInit() {
  }
  @HostListener('click', ['$event'])
  _click(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    this.onClick.emit(this);
    this.active = !this.active;
  }
}