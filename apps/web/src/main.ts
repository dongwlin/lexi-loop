import { createApp } from 'vue'
import './lib/api'
import './styles/main.css'
import App from './app/App.vue'
import { installApp } from './app/providers'
import { useThemeStore } from '@/stores/theme'

const app = createApp(App)
installApp(app)
// 首屏绘制前解析并应用主题（设计系统方案 §11）；需在 pinia 安装后、挂载前执行。
useThemeStore().init()
app.mount('#app')
