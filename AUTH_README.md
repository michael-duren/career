# Authentication Setup for Netlify

This Astro application now includes username/password authentication for protecting your content on Netlify.

## Features

- Username and password login system
- JWT-based authentication
- Server-side route protection via middleware
- Netlify Functions for authentication backend
- Automatic redirect to login for unauthenticated users
- Logout functionality

## Setup Instructions

### 1. Environment Variables

Set up the following environment variables in your Netlify dashboard:

1. Go to your Netlify site dashboard
2. Navigate to Site Settings → Environment Variables
3. Add these variables:

```
JWT_SECRET=<generate-a-secure-random-string>
AUTH_USERNAME=<your-username>
AUTH_PASSWORD_HASH=<bcrypt-hash-of-your-password>
```

### 2. Generate Password Hash

For security, passwords are stored as bcrypt hashes. To generate a hash:

Option A: Use an online bcrypt generator:
- Visit https://bcrypt-generator.com/
- Enter your desired password
- Use cost factor 10
- Copy the generated hash

Option B: Use Node.js:
```javascript
const bcrypt = require('bcryptjs');
const password = 'your-password-here';
const hash = bcrypt.hashSync(password, 10);
console.log(hash);
```

### 3. Required Credentials

Set `JWT_SECRET`, `AUTH_USERNAME`, and `AUTH_PASSWORD_HASH` in the Netlify Functions environment. Missing configuration disables deployed login. Locally, `npm run dev` supplies missing settings with username `admin`, password `local-dev-password`, and a generated session signing secret. Shell and `.env` settings take precedence. See [the interactive workspace setup](README.md#netlify-setup).

## How It Works

1. **Authentication Flow:**
   - User visits any protected page
   - Middleware checks for authentication cookie
   - If not authenticated, redirects to `/login`
   - User enters credentials
   - Credentials verified via Netlify Function
   - JWT token generated and stored in cookies
   - User redirected to original page

2. **Protected Routes:**
   - All routes except `/login` are protected by default
   - Edit `src/middleware.ts` to customize public routes

3. **Netlify Function:**
   - Located at `netlify/functions/auth.js`
   - Handles login and token verification
   - Returns JWT tokens valid for 24 hours

## Security Considerations

1. **Use HTTPS:** Netlify provides HTTPS by default
2. **Strong JWT Secret:** Generate a long, random string
3. **Secure Password:** Use a strong password and never commit it to git
4. **Environment Variables:** Never hardcode credentials
5. **Regular Updates:** Rotate JWT secret periodically

## Testing Locally

Run `npm run dev`, then sign in with `admin` / `local-dev-password`. No `.env` file
is required. To use custom credentials, set the three authentication variables in
your shell or `.env`. Restarting the server requires signing in again when using
the generated signing secret. `netlify dev` requires explicitly configured credentials.

## Customization

### Changing Login Page Design
Edit `src/pages/login.astro` to customize the login interface.

### Adding More Users
Modify `netlify/functions/auth.js` to check against a database or user list instead of single credentials.

### Session Duration
Change token expiration in `netlify/functions/auth.js`:
```javascript
{ expiresIn: '24h' } // Change to desired duration
```

## Troubleshooting

1. **"Invalid credentials" error:**
   - Check username and password
   - Verify environment variables are set in Netlify

2. **Redirect loop:**
   - Clear browser cookies
   - Check middleware configuration

3. **Function not found:**
   - Ensure `netlify/functions` directory exists
   - Check `netlify.toml` configuration

## Support

For issues or questions, please check the documentation or open an issue in the repository.