
import { DataSource, DataSourceOptions } from 'typeorm';
import request from 'supertest';
import App from '../../src/app';
import { User } from '../../src/entities/User';

import { PostgreSqlContainer, StartedPostgreSqlContainer } from '@testcontainers/postgresql';
import { Client } from 'pg';
import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import { initializeDatabase } from '../../src/config/database';

describe('User Routes Integration', () => {
    let postgresContainer: StartedPostgreSqlContainer;
    let app: App;


    beforeAll(async () => {
        // Start PostgreSQL container
        postgresContainer = await new PostgreSqlContainer("postgres:latest")
            .withDatabase('testdb')
            .withUsername('testuser')
            .withPassword('testpass')
            .start();

        // Database connection configuration
        const connectionOptions: DataSourceOptions = {
            type: 'postgres',
            host: postgresContainer.getHost(),
            port: postgresContainer.getPort(),
            username: postgresContainer.getUsername(),
            password: postgresContainer.getPassword(),
            database: postgresContainer.getDatabase(),
            entities: [User],
            synchronize: true,
            logging: false
        };

        // Initialize application with database
        app = new App();
        await initializeDatabase(connectionOptions);
    });

    afterAll(async () => {
        await postgresContainer.stop();
    });

    it('should create a new user', async () => {
        const id = Math.random() * 100
        const userData = {
            name: 'Jane Doe',
            email: `jane${id}@example.com`
        };

        const response = await request(app.getApp())
            .post('/users')
            .send(userData)
            .expect(201);

        expect(response.body.name).toBe(userData.name);
        expect(response.body.email).toBe(userData.email);
        expect(response.body.id).toBeDefined();
    });
    it('should not create a user with missing required fields', async () => {
        const userData = {
            name: 'John Doe'
        };

        const response = await request(app.getApp())
            .post('/users')
            .send(userData)
            .expect(500);

        expect(response.body).toHaveProperty('message');
    });
    it('should not create a user with a duplicate email', async () => {
        const id = Math.random() * 100;
        const userData = {
            name: 'Jane Doe',
            email: `jane${id}@example.com`
        };

        await request(app.getApp())
            .post('/users')
            .send(userData)
            .expect(201);

        const response = await request(app.getApp())
            .post('/users')
            .send(userData)
            .expect(500);

        expect(response.body).toHaveProperty('message');
    });

    it('should retrieve users', async () => {
        const id = Math.random() * 100
        const userData = {
            name: 'Jane Doe',
            email: `jane${id}@example.com`
        };

        // Create a user first
        await request(app.getApp())
            .post('/users')
            .send(userData)


        const response = await request(app.getApp())
            .get('/users')
            .expect(200);

        expect(response.body.length).toBeGreaterThan(0);
        expect(response.body.some((user: User) => user.email === userData.email)).toBeTruthy();
    });
});
