import { DataSource, DataSourceOptions } from "typeorm";


export let dataSource: DataSource;
export const initializeDatabase = async (dbConfig: DataSourceOptions,) => {
    try {
        dataSource = new DataSource(dbConfig);
        await dataSource.initialize();
        console.log("Database connection established");
    } catch (error) {
        console.error("Error initializing database", error);
        process.exit(1);
    }
};
