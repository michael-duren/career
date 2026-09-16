import { randomBytes } from 'node:crypto';
import bcrypt from 'bcryptjs';

// Only the local dev launcher calls this; deployed auth still requires configuration.
export function configureDevAuth(env) {
  const defaults = [];
  if (!env.JWT_SECRET) {
    env.JWT_SECRET = randomBytes(32).toString('hex');
    defaults.push('JWT_SECRET (generated for this session)');
  }
  if (!env.AUTH_USERNAME) {
    env.AUTH_USERNAME = 'admin';
    defaults.push('AUTH_USERNAME=admin');
  }
  if (!env.AUTH_PASSWORD_HASH || env.AUTH_PASSWORD_HASH === '$2a$10$YourHashedPasswordHere') {
    env.AUTH_PASSWORD_HASH = bcrypt.hashSync('local-dev-password', 10);
    defaults.push('password=local-dev-password');
  }
  return defaults;
}
