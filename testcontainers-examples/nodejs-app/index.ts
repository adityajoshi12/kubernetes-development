import App from './src/app';
import { DataSourceOptions } from 'typeorm';
import { initializeDatabase } from './src/config/database';
import { User } from './src/entities/User';

async function startServer() {

    try {
        // Initialize the application
        const app = new App();

        const dbConfig: DataSourceOptions = {
            type: 'postgres',
            host: process.env.DB_HOST || '0.0.0.0',
            port: parseInt(process.env.DB_PORT || '5432'),
            username: process.env.DB_USER || 'postgres',
            password: process.env.DB_PASSWORD || 'mysecretpassword',
            database: process.env.DB_NAME || 'postgres',
            entities: [User],
            synchronize: true,
            logging: process.env.NODE_ENV === 'development'
        };

        // Initialize database connection
        await initializeDatabase(dbConfig);

        // Start the server
        const PORT = process.env.PORT || 3000;
        app.getApp().listen(PORT, () => {
            console.log(`Server running on port ${PORT}`);
        });
    } catch (error) {
        console.error('Failed to start server:', error);
        process.exit(1);
    }
}

// Run the server
startServer();
