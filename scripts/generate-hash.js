import { hashSync } from "bcryptjs";
import readline from "readline";

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
});

rl.question("enter pwd to hash: ", (password) => {
  const hash = hashSync(password, 10);
  console.log("\nYour password hash is: ");
  console.log(hash);
  console.log("\nCopy this hash and use it for AUTH_PASSWORD_HASH in Netlify");

  rl.close(); 
});
