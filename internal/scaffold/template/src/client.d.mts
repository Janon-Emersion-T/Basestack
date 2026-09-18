export interface User { id: string; email: string; emailVerified: boolean; status: string; createdAt: string }
export interface ObjectMetadata { id: string; size: number; modifiedAt: string }
export interface Client {
  auth: {
    signup(email: string, password: string): Promise<{ user: User }>;
    login(email: string, password: string): Promise<User>;
    current(): Promise<User>;
    logout(): Promise<{ loggedOut: boolean }>;
    verifyEmail(token: string): Promise<{ user: User }>;
    requestVerification(email: string): Promise<{ message: string }>;
    forgotPassword(email: string): Promise<{ message: string }>;
    resetPassword(token: string, password: string): Promise<{ passwordChanged: boolean }>;
  };
  functions: { run<T = unknown>(name: string, input?: unknown): Promise<T> };
  storage: {
    list(bucket: string): Promise<ObjectMetadata[]>;
    upload(bucket: string, blob: Blob): Promise<ObjectMetadata>;
    download(bucket: string, id: string): Promise<Blob>;
    delete(bucket: string, id: string): Promise<{ deleted: boolean }>;
  };
}
export function createClient(config: { schemaVersion: number; apiURL: string }, transport?: typeof fetch): Client;
