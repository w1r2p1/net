import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { FlexLayoutModule } from '@angular/flex-layout';
import { AppComponent } from './app.component';
import {
  ImViewComponent,
  ImRecentBarComponent,
  ImRecentItemComponent,
  ImHeadComponent,
  ImHistoryViewComponent,
  ImHistoryMessageComponent,
  CreateChatDialogComponent,
  AlertDialogComponent,
  ImInfoDialogComponent
} from '../components';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { SocketService, UserService } from '../providers';
import { ToolService } from '../providers/tool/tool.service';
import {
  MdCheckboxModule,
  MdMenuModule,
  MdIconModule,
  MdDialogModule,
  MdInputModule,
  MdButtonModule,
} from '@angular/material';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import 'web-animations-js'
import { PerfectScrollbarModule } from 'ngx-perfect-scrollbar';
import { PerfectScrollbarConfigInterface } from 'ngx-perfect-scrollbar';
import { HttpModule } from '@angular/http';
import { EditorModule } from '../components/editor/editor.module'

const PERFECT_SCROLLBAR_CONFIG: PerfectScrollbarConfigInterface = {
  suppressScrollX: true
};

@NgModule({
  declarations: [
    AppComponent,
    ImViewComponent,
    ImRecentBarComponent,
    ImRecentItemComponent,
    ImHeadComponent,
    ImHistoryViewComponent,
    ImHistoryMessageComponent,
    CreateChatDialogComponent,
    AlertDialogComponent,
    ImInfoDialogComponent,
  ],
  imports: [
    HttpModule,
    FormsModule,
    ReactiveFormsModule,
    BrowserModule,
    FlexLayoutModule,
    BrowserAnimationsModule,
    MdCheckboxModule,
    MdMenuModule,
    MdIconModule,
    MdDialogModule,
    MdInputModule,
    MdButtonModule,
    PerfectScrollbarModule.forRoot(PERFECT_SCROLLBAR_CONFIG),
    EditorModule,
  ],
  entryComponents: [
    CreateChatDialogComponent,
    AlertDialogComponent,
    ImInfoDialogComponent
  ],
  providers: [SocketService, UserService, ToolService],
  bootstrap: [AppComponent]
})
export class AppModule { }