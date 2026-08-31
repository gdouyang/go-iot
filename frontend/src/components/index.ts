import type { App } from 'vue'
import { Icon } from './Icon'
import { BaseButton } from './Button'
import { PageTable } from './Table'
import { Dialog } from './Dialog'
import { ElTextarea } from './Input'
import { ContentWrap } from './ContentWrap'
import { ContentDetailWrap } from './ContentDetailWrap'
import { Echart } from './Echart'
import { AceEditor } from './AceEditor'
export const setupGlobCom = (app: App<Element>): void => {
  app.component('Icon', Icon)
  app.component('BaseButton', BaseButton)
  app.component('PageTable', PageTable)
  app.component('Dialog', Dialog)
  app.component('ElTextarea', ElTextarea)
  app.component('ContentWrap', ContentWrap)
  app.component('ContentDetailWrap', ContentDetailWrap)
  app.component('Echart', Echart)
  app.component('AceEditor', AceEditor)
}
