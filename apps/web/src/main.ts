import { createApp } from 'vue'
import './lib/api'
import './styles/main.css'
import App from './app/App.vue'
import { installApp } from './app/providers'

const app = createApp(App)
installApp(app)
app.mount('#app')
