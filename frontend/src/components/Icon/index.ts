import { addCollection } from '@iconify/vue'
import iconfile from './icon.json'

try {
  if (window.localStorage) {
    const localStorage = window.localStorage
    if (!localStorage.getItem('iconify0')) {
      localStorage.setItem('iconify-version', 'iconify2')
      localStorage.setItem('iconify-count', '12')
      for (const key in iconfile) {
        const element = iconfile[key]
        addCollection(element.data)
        localStorage.setItem(key, JSON.stringify(element))
      }
    }
  }
} catch (error) {
  console.error(error)
}

import Icon from './src/Icon.vue'
export type { IconTypes } from './src/types'

export { Icon }
