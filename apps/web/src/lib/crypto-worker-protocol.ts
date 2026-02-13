export interface GenerateRegistrationDataRequest {
  type: "generateRegistrationData";
  password: string;
}

export interface DeriveLoginKeysRequest {
  type: "deriveLoginKeys";
  password: string;
  saltHex: string;
}

export interface DecryptVaultKeyRequest {
  type: "decryptVaultKey";
  masterKey: Uint8Array;
  encryptedVaultKeyHex: string;
  nonceHex: string;
}

export interface CreatePasteRequest {
  type: "createPaste";
  title: string;
  body: string;
  vaultKey: Uint8Array;
}

export interface EncryptFileRequest {
  type: "encryptFile";
  file: Uint8Array;
  key: Uint8Array;
  aad: Uint8Array;
}

export interface DecryptFileRequest {
  type: "decryptFile";
  ciphertext: Uint8Array;
  nonce: Uint8Array;
  key: Uint8Array;
  aad: Uint8Array;
}

export type CryptoRequest =
  | GenerateRegistrationDataRequest
  | DeriveLoginKeysRequest
  | DecryptVaultKeyRequest
  | CreatePasteRequest
  | EncryptFileRequest
  | DecryptFileRequest;

export interface GenerateRegistrationDataResponse {
  type: "generateRegistrationData";
  salt: string;
  authKey: string;
  encryptedVaultKey: string;
  vaultKeyNonce: string;
}

export interface DeriveLoginKeysResponse {
  type: "deriveLoginKeys";
  masterKey: Uint8Array;
  authKey: string;
}

export interface DecryptVaultKeyResponse {
  type: "decryptVaultKey";
  vaultKey: Uint8Array;
}

export interface CreatePasteResponse {
  type: "createPaste";
  encryptedTitleCiphertext: Uint8Array;
  encryptedTitleNonce: Uint8Array;
  encryptedBodyCiphertext: Uint8Array;
  encryptedBodyNonce: Uint8Array;
  encryptedPasteKeyCiphertext: Uint8Array;
  encryptedPasteKeyNonce: Uint8Array;
  pasteKeyBase64Url: string;
  pasteKey: Uint8Array;
}

export interface EncryptFileResponse {
  type: "encryptFile";
  ciphertext: Uint8Array;
  nonce: Uint8Array;
}

export interface DecryptFileResponse {
  type: "decryptFile";
  plaintext: Uint8Array;
}

export type CryptoResponse =
  | GenerateRegistrationDataResponse
  | DeriveLoginKeysResponse
  | DecryptVaultKeyResponse
  | CreatePasteResponse
  | EncryptFileResponse
  | DecryptFileResponse;

// Envelope types

export interface WorkerRequest {
  id: number;
  request: CryptoRequest;
}

export interface WorkerResponseOk {
  id: number;
  ok: true;
  response: CryptoResponse;
}

export interface WorkerResponseError {
  id: number;
  ok: false;
  error: string;
}

export type WorkerResponse = WorkerResponseOk | WorkerResponseError;
