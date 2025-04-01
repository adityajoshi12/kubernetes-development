mkdir nodejs-redis-app
cd nodejs-redis-app
npm init -y



echo '{
  "name": "nodejs-redis-app",
  "version": "1.0.0",
  "main": "index.js",
  "type": "module",
  "scripts": {
    "test": "echo \"Error: no test specified\" && exit 1"
  },
  "keywords": [],
  "author": "",
  "license": "ISC",
  "description": "",
  "dependencies": {
    "redis": "^4.7.0"
  }
}' > package.json



echo '
import { createClient } from 'redis';

const client = await createClient({
          url:"redis://nodejs-env-database"
})
.on('error', err => console.log('Redis Client Error', err))
.connect();

await client.set('key', 'value');
const value = await client.get('key');
console.log(value)
await client.disconnect();
' > app.js

npm install redis
