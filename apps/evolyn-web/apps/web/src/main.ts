import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import '~/styles/index.scss'
import '@evolyn.do/ui/style.css'
import 'element-plus/theme-chalk/src/message.scss'
import 'element-plus/theme-chalk/src/message-box.scss'
import 'element-plus/theme-chalk/src/overlay.scss' // the modal class for message box

const app = createApp(App)

app.use(router)

app.mount('#app')
