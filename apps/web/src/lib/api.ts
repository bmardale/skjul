import ky, { type HTTPError } from "ky";

export interface RegisterRequest {
  username: string;
  authKey: string;
  salt: string;
  encryptedVaultKey: string;
  vaultKeyNonce: string;
}

export interface RegisterResponse {
  id: string;
}

export interface LoginChallengeRequest {
  username: string;
}

export interface LoginChallengeResponse {
  salt: string;
}

export interface LoginRequest {
  username: string;
  authKey: string;
}

export interface LoginResponse {
  user_id: string;
  username: string;
}

export interface MeResponse {
  user_id: string;
  username: string;
  created_at: string;
  salt: string;
  encryptedVaultKey: string;
  vaultKeyNonce: string;
}

export interface SessionResponse {
  id: string;
  created_at: string;
  expires_at: string;
  current: boolean;
}

export interface ApiError {
  code: string;
  message: string;
}

const client = ky.create({
  prefixUrl: import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080",
  credentials: "include",
  timeout: 15_000,
  hooks: {
    beforeError: [
      async (error: HTTPError) => {
        const body = await error.response
          .json<ApiError>()
          .catch(() => undefined);
        if (body) {
          error.message = body.message;
          (error as HTTPError & { apiError: ApiError }).apiError = body;
        }
        return error;
      },
    ],
  },
});

export function getApiError(err: unknown): ApiError | undefined {
  return (err as HTTPError & { apiError?: ApiError })?.apiError;
}

export const api = {
  register(data: RegisterRequest) {
    return client.post("api/v1/auth/register", { json: data }).json<RegisterResponse>();
  },

  loginChallenge(data: LoginChallengeRequest) {
    return client
      .post("api/v1/auth/login/challenge", { json: data })
      .json<LoginChallengeResponse>();
  },

  login(data: LoginRequest) {
    return client.post("api/v1/auth/login", { json: data }).json<LoginResponse>();
  },

  logout() {
    return client.post("api/v1/auth/logout").json<void>();
  },

  me() {
    return client.get("api/v1/me").json<MeResponse>();
  },

  listSessions() {
    return client.get("api/v1/sessions").json<SessionResponse[]>();
  },

  revokeSession(id: string) {
    return client.delete(`api/v1/sessions/${id}`).json<void>();
  },

  deleteAccount() {
    return client.delete("api/v1/me").json<void>();
  },
} as const;
