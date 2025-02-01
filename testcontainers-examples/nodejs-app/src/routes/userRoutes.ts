import { Router, Request, Response } from 'express';
import { DataSource } from 'typeorm';
import { User } from '../entities/User';
import { dataSource } from '../config/database';


const router = Router();

router.post('/', async (req: Request, res: Response) => {
    try {

        const userRepository = dataSource.getRepository(User);
        const newUser = userRepository.create(req.body);
        const savedUser = await userRepository.save(newUser);
        res.status(201).json(savedUser);
    } catch (error) {
        res.status(500).json({ message: (error as Error).message });
    }
});

router.get('/', async (req: Request, res: Response) => {
    try {
        const userRepository = dataSource.getRepository(User);
        const users = await userRepository.find();
        res.json(users);
    } catch (error) {
        res.status(500).json({ message: (error as Error).message });
    }
});

export default router;
