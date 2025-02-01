# Testcontainers Node.js Example
Testcontainers is an open source library for providing throwaway, lightweight instances of databases, message brokers, web browsers, or just about anything that can run in a Docker container.

### Create User
```bash
curl -X POST http://localhost:3000/users -H "Content-Type: application/json" -d '{"name":"aditya","email":"abc@example.com"}'
```


### Get All User
```bash
curl -X GET http://localhost:3000/users -H "Content-Type: application/json"
```
