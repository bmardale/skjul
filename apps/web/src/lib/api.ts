import ky, { type HTTPError } from "ky";

export interface RegisterRequest {
  username: string;
  auth_key: string;
  salt: string;
  encrypted_vault_key: string;
  vault_key_nonce: string;
  invite_code?: string;
}

export interface PublicConfig {
  require_invite_code: boolean;
}

export interface Invitation {
  id: string;
  code: string;
  used: boolean;
  created_at: string;
  used_at?: string;
}

export interface ListInvitationsResponse {
  remaining_quota: number;
  invitations: Invitation[];
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
  auth_key: string;
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
  encrypted_vault_key: string;
  vault_key_nonce: string;
  is_admin: boolean;
}

export interface AdminUserListItem {
  id: string;
  username: string;
  invite_quota: number;
  created_at: string;
}

export interface AdminUserDetail {
  id: string;
  username: string;
  invite_quota: number;
  created_at: string;
  paste_count: number;
  total_attachment_size: number;
}

export interface SessionResponse {
  id: string;
  created_at: string;
  expires_at: string;
  current: boolean;
}

export interface CreatePasteRequest {
  encrypted_title_ciphertext: string;
  encrypted_title_nonce: string;
  encrypted_body_ciphertext: string;
  encrypted_body_nonce: string;
  encrypted_paste_key_ciphertext: string;
  encrypted_paste_key_nonce: string;
  expiration: string;
  burn_after_reading: boolean;
  language_id?: string;
}

export interface CreatePasteResponse {
  id: string;
  created_at: string;
  expires_at: string;
}

export interface PasteAttachment {
  id: string;
  encrypted_size: number;
  filename_ciphertext: string;
  filename_nonce: string;
  content_nonce: string;
  mime_ciphertext: string;
  mime_nonce: string;
  download_url: string;
}

export interface GetPasteResponse {
  id: string;
  burn_after_read: boolean;
  title_ciphertext: string;
  title_nonce: string;
  body_ciphertext: string;
  body_nonce: string;
  encrypted_paste_key_ciphertext: string;
  encrypted_paste_key_nonce: string;
  created_at: string;
  expires_at: string;
  language_id?: string;
  attachments: PasteAttachment[];
}

export interface CreateAttachmentRequest {
  encrypted_size: number;
  filename_ciphertext: string;
  filename_nonce: string;
  content_nonce: string;
  mime_ciphertext: string;
  mime_nonce: string;
}

export interface CreateAttachmentResponse {
  id: string;
  upload_url: string;
}

export interface PasteListItem {
  id: string;
  burn_after_read: boolean;
  title_ciphertext: string;
  title_nonce: string;
  encrypted_paste_key_ciphertext: string;
  encrypted_paste_key_nonce: string;
  created_at: string;
  expires_at: string;
  language_id?: string;
  attachment_count: number;
}

export interface ListPastesResponse {
  items: PasteListItem[];
  next_cursor?: string;
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

export function getRateLimitMessage(err: unknown): string | undefined {
  const apiErr = getApiError(err);
  if (apiErr?.code !== "RATE_LIMITED") return undefined;
  const httpErr = err as HTTPError & { response?: Response };
  const retryAfter = httpErr.response?.headers?.get("Retry-After");
  if (retryAfter) {
    const sec = parseInt(retryAfter, 10);
    if (!isNaN(sec)) return `Too many requests. Please try again in ${sec} seconds.`;
  }
  return "Too many requests. Please try again later.";
}

export const api = {
  getPublicConfig() {
    return client.get("api/v1/config").json<PublicConfig>();
  },

  register(data: RegisterRequest) {
    return client.post("api/v1/auth/register", { json: data }).json<RegisterResponse>();
  },

  generateInvite() {
    return client.post("api/v1/invitations").json<{ code: string }>();
  },

  listInvites() {
    return client.get("api/v1/invitations").json<ListInvitationsResponse>();
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

  createPaste(data: CreatePasteRequest) {
    return client.post("api/v1/pastes", { json: data }).json<CreatePasteResponse>();
  },

  createAttachment(pasteId: string, data: CreateAttachmentRequest) {
    return client
      .post(`api/v1/pastes/${pasteId}/attachments`, { json: data })
      .json<CreateAttachmentResponse>();
  },

  getPaste(id: string) {
    return client.get(`api/v1/pastes/${id}`).json<GetPasteResponse>();
  },

  async uploadToPresignedUrl(url: string, encryptedBytes: Uint8Array): Promise<void> {
    const body = new Uint8Array(encryptedBytes.length);
    body.set(encryptedBytes);
    const res = await fetch(url, {
      method: "PUT",
      body,
      headers: {
        "Content-Type": "application/octet-stream",
        "Content-Length": String(encryptedBytes.length),
      },
    });
    if (!res.ok) {
      throw new Error(`Upload failed: ${res.status} ${res.statusText}`);
    }
  },

  listPastes(cursor?: string) {
    const searchParams = cursor ? { cursor } : undefined;
    return client.get("api/v1/pastes", { searchParams }).json<ListPastesResponse>();
  },

  deletePaste(id: string) {
    return client.delete(`api/v1/pastes/${id}`).json<void>();
  },

  adminListUsers() {
    return client.get("api/v1/admin/users").json<AdminUserListItem[]>();
  },

  adminGetUser(id: string) {
    return client.get(`api/v1/admin/users/${id}`).json<AdminUserDetail>();
  },

  adminDeleteUser(id: string) {
    return client.delete(`api/v1/admin/users/${id}`).then(() => undefined);
  },

  adminUpdateInviteQuota(id: string, quota: number) {
    return client
      .patch(`api/v1/admin/users/${id}/invite-quota`, { json: { quota } })
      .then(() => undefined);
  },
} as const;
