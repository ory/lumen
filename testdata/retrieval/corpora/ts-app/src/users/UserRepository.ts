export interface User {
  id: string;
  email: string;
  displayName: string;
  disabledAt?: Date;
}

export interface UserRepository {
  findById(id: string): Promise<User | undefined>;
  findByEmail(email: string): Promise<User | undefined>;
  save(user: User): Promise<void>;
}
