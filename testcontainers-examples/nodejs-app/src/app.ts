import express, { Application } from 'express';
import { DataSource, DataSourceOptions } from 'typeorm';
import { User } from './entities/User';
import userRoutes from './routes/userRoutes';
import { initializeDatabase } from './config/database';

class App {
  public app: Application;


  constructor() {
    this.app = express();
    this.app.use(express.json());
    this.app.use('/users', userRoutes);

  }
  getApp(): Application {
    return this.app;
  }
}

export default App;
