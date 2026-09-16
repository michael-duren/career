import bcrypt from 'bcryptjs';

console.log('=== Authentication Test Script ===\n');

// Test the default password
const defaultPassword = 'password123';
const testHash = bcrypt.hashSync(defaultPassword, 10);

console.log('Default password:', defaultPassword);
console.log('Generated hash:', testHash);
console.log('Verification:', bcrypt.compareSync(defaultPassword, testHash) ? '✅ PASS' : '❌ FAIL');

console.log('\n--- Testing Your Password ---\n');

// Change this to your desired password
const yourPassword = 'YourPasswordHere';  // <-- CHANGE THIS

if (yourPassword === 'YourPasswordHere') {
  console.log('⚠️  Please edit this file and set your password first!\n');
} else {
  const yourHash = bcrypt.hashSync(yourPassword, 10);

  console.log('Your password:', yourPassword);
  console.log('Your hash:', yourHash);
  console.log('\n📋 Copy this hash for AUTH_PASSWORD_HASH in Netlify:');
  console.log(yourHash);

  // Test verification
  console.log('\nVerification test:', bcrypt.compareSync(yourPassword, yourHash) ? '✅ PASS' : '❌ FAIL');
}

console.log('\n--- Environment Variables for Netlify ---\n');
console.log('AUTH_USERNAME=admin');
console.log('AUTH_PASSWORD_HASH=' + (yourPassword === 'YourPasswordHere' ? '<your-hash-here>' : yourHash));
console.log('JWT_SECRET=' + require('crypto').randomBytes(32).toString('hex'));