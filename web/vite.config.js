import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
export default defineConfig({plugins:[react()],server:{host:'0.0.0.0',proxy:{'/api/menu':{target:'http://localhost:8001',changeOrigin:true},'/api/orders':{target:'http://localhost:8002',changeOrigin:true},'/api/stats':{target:'http://localhost:8004',changeOrigin:true},'/api/health':{target:'http://localhost:8004',changeOrigin:true},'/health':{target:'http://localhost:8001',changeOrigin:true},'/info':{target:'http://localhost:8002',changeOrigin:true}}}});
