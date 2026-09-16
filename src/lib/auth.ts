interface AuthResponse {
  authenticated: boolean;
  username?: string;
  error?: string;
}

interface VerifyResponse {
  authenticated: boolean;
  username?: string;
  error?: string;
}

const AUTH_TOKEN_KEY = "auth_token";
const AUTH_USERNAME_KEY = "auth_username";

export class AuthService {
  static getUsername(): string | null {
    if (typeof window === "undefined") return null;
    return localStorage.getItem(AUTH_USERNAME_KEY);
  }

  static setAuth(username: string): void {
    if (typeof window === "undefined") return;
    localStorage.setItem(AUTH_USERNAME_KEY, username);
  }

  static clearAuth(): void {
    if (typeof window === "undefined") return;
    localStorage.removeItem(AUTH_TOKEN_KEY);
    localStorage.removeItem(AUTH_USERNAME_KEY);
  }

  static async login(
    username: string,
    password: string,
  ): Promise<AuthResponse> {
    try {
      this.clearAuth();
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          username,
          password,
        }),
      });

      const data: AuthResponse = await response.json();

      if (data.authenticated && data.username) {
        this.setAuth(data.username);
      }

      return data;
    } catch (error) {
      return {
        authenticated: false,
        error: "Network error. Please try again.",
      };
    }
  }

  static async verify(): Promise<VerifyResponse> {
    try {
      const response = await fetch("/api/auth/verify", { cache: "no-store" });

      const data: VerifyResponse = await response.json();

      if (!data.authenticated) {
        this.clearAuth();
      }

      return data;
    } catch (error) {
      this.clearAuth();
      return { authenticated: false };
    }
  }

  static async logout(): Promise<void> {
    await fetch('/api/auth/logout', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' }).catch(() => undefined);
    this.clearAuth();
    window.location.href = "/login";
  }

  static isAuthenticated(): boolean {
    return false;
  }
}

// Server-side authentication check for Astro pages
export async function checkAuth(
  request: Request,
): Promise<{ authenticated: boolean; username?: string }> {
  const cookieHeader = request.headers.get("cookie");

  if (!cookieHeader) {
    return { authenticated: false };
  }

  // Parse cookies to get the token
  const cookies = Object.fromEntries(
    cookieHeader.split("; ").map((cookie) => {
      const [key, value] = cookie.split("=");
      return [key, decodeURIComponent(value || "")];
    }),
  );

    const token = cookies.session;

  if (!token) {
    return { authenticated: false };
  }

  try {
    // Verify token with the backend
    const response = await fetch(
      `${process.env.URL || "http://localhost:8080"}/api/auth/verify`,
      {
        headers: { Cookie: `session=${encodeURIComponent(token)}` },
      },
    );

    const data: VerifyResponse = await response.json();

    return {
      authenticated: data.authenticated || false,
      username: data.username,
    };
  } catch (error) {
    return { authenticated: false };
  }
}
