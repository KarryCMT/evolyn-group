import type { UserModule } from './types'
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'

// import "~/styles/element/index.scss";

// import ElementPlus from "element-plus";
// import all element css, uncommented next line
// import "element-plus/dist/index.css";

// or use cdn, uncomment cdn link in `index.html`

import '~/styles/index.scss'

// If you want to use ElMessage, import it.
import 'element-plus/theme-chalk/src/message.scss'
import 'element-plus/theme-chalk/src/message-box.scss'
import 'element-plus/theme-chalk/src/overlay.scss' // the modal class for message box

const app = createApp(App)

// 路由先于业务模块安装，保证模块初始化时 router 已可用
app.use(router)

// install all modules under `modules/`
Object.values(import.meta.glob<{ install: UserModule }>('./modules/*.ts', { eager: true }))
  .forEach(i => i.install?.(app))

app.mount('#app')
