import { createApp } from 'vue'
import App from './App.vue'
import router from './router'

import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

import JsonViewer from 'vue-json-viewer'
import 'vue-json-viewer/style.css'

import "/src/assets/jsoneditor.min.js"
import "/src/assets/bootstrap.min.css"
JSONEditor.defaults.options.theme = "bootstrap3";

const app = createApp(App)

app.use(JsonViewer)
app.use(ElementPlus)
app.use(router)

app.mount('#app')

