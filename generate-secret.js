import { randomBytes } from 'crypto';

const secret = randomBytes(32).toString('hex');
console.log('\nYour JWT Secret is:');
console.log(secret);
console.log('\nCopy this and use it for JWT_SECRET in Netlify');
