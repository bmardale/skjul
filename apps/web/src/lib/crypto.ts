import { hexToBytes, randomBytes } from "@noble/hashes/utils.js";
import { hmac } from "@noble/hashes/hmac.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { xchacha20poly1305 } from "@noble/ciphers/chacha.js";
import type {
  CryptoRequest,
  CryptoResponse,
  WorkerRequest,
  WorkerResponse,
  GenerateRegistrationDataResponse,
  DeriveLoginKeysResponse,
  DecryptVaultKeyResponse,
  CreatePasteResponse,
  EncryptFileResponse,
  DecryptFileResponse,
} from "./crypto-worker-protocol";

const encoder = new TextEncoder();

function deriveSubkey(key: Uint8Array, context: string): Uint8Array {
  return hmac(sha256, key, encoder.encode(context));
}

export const AAD = {
  TITLE: encoder.encode("skjul:v1:title"),
  BODY: encoder.encode("skjul:v1:body"),
  PASTE_KEY: encoder.encode("skjul:v1:paste_key"),
  FILE: encoder.encode("skjul:v1:file"),
  FILENAME: encoder.encode("skjul:v1:filename"),
  MIME: encoder.encode("skjul:v1:mime"),
  VAULT_KEY: encoder.encode("skjul:v1:vault_key"),
} as const;

export function derivePasteSubkeys(pasteKey: Uint8Array) {
  return {
    contentKey: deriveSubkey(pasteKey, "paste_content_v1"),
    metaKey: deriveSubkey(pasteKey, "paste_meta_v1"),
    fileKey: deriveSubkey(pasteKey, "paste_file_v1"),
  };
}

export function base64UrlToBytes(str: string): Uint8Array {
  let base64 = str.replace(/-/g, "+").replace(/_/g, "/");
  while (base64.length % 4) base64 += "=";
  const binary = atob(base64);
  return Uint8Array.from(binary, (c) => c.charCodeAt(0));
}

let worker: Worker | null = null;
let nextId = 0;
const pending = new Map<
  number,
  { resolve: (value: CryptoResponse) => void; reject: (error: Error) => void }
>();

function getWorker(): Worker {
  if (!worker) {
    worker = new Worker(
      new URL("./crypto-worker.ts", import.meta.url),
      { type: "module" },
    );
    worker.onmessage = (event: MessageEvent<WorkerResponse>) => {
      const { id, ...rest } = event.data;
      const entry = pending.get(id);
      if (!entry) return;
      pending.delete(id);
      if (rest.ok) {
        entry.resolve(rest.response);
      } else {
        entry.reject(new Error(rest.error));
      }
    };
    worker.onerror = (event) => {
      const err = new Error(`Worker error: ${event.message}`);
      for (const [id, entry] of pending) {
        entry.reject(err);
        pending.delete(id);
      }
    };
  }
  return worker;
}

function postToWorker<T extends CryptoResponse>(
  request: CryptoRequest,
  transfer: Transferable[] = [],
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const id = nextId++;
    pending.set(id, {
      resolve: resolve as (value: CryptoResponse) => void,
      reject,
    });
    const msg: WorkerRequest = { id, request };
    getWorker().postMessage(msg, transfer);
  });
}

interface RegistrationResult {
    salt: string;
    authKey: string;
    encryptedVaultKey: string;
    vaultKeyNonce: string;
}

export async function generateRegistrationData(password: string): Promise<RegistrationResult> {
  const response = await postToWorker<GenerateRegistrationDataResponse>({
    type: "generateRegistrationData",
    password,
  });
  return {
    salt: response.salt,
    authKey: response.authKey,
    encryptedVaultKey: response.encryptedVaultKey,
    vaultKeyNonce: response.vaultKeyNonce,
  };
}

export async function deriveLoginKeys(
  password: string,
  saltHex: string,
): Promise<{ masterKey: Uint8Array; authKey: string }> {
  const response = await postToWorker<DeriveLoginKeysResponse>({
    type: "deriveLoginKeys",
    password,
    saltHex,
  });
  return { masterKey: response.masterKey, authKey: response.authKey };
}

export async function decryptVaultKey(
  masterKey: Uint8Array,
  encryptedVaultKeyHex: string,
  nonceHex: string,
): Promise<Uint8Array> {
  const response = await postToWorker<DecryptVaultKeyResponse>({
    type: "decryptVaultKey",
    masterKey,
    encryptedVaultKeyHex,
    nonceHex,
  });
  return response.vaultKey;
}

export interface PasteResult {
  encryptedTitleCiphertext: Uint8Array;
  encryptedTitleNonce: Uint8Array;
  encryptedBodyCiphertext: Uint8Array;
  encryptedBodyNonce: Uint8Array;
  encryptedPasteKeyCiphertext: Uint8Array;
  encryptedPasteKeyNonce: Uint8Array;
  pasteKeyBase64Url: string;
  pasteKey: Uint8Array;
}

export async function createPaste(title: string, body: string, vaultKey: Uint8Array): Promise<PasteResult> {
  const response = await postToWorker<CreatePasteResponse>({
    type: "createPaste",
    title,
    body,
    vaultKey,
  });
  return {
    encryptedTitleCiphertext: response.encryptedTitleCiphertext,
    encryptedTitleNonce: response.encryptedTitleNonce,
    encryptedBodyCiphertext: response.encryptedBodyCiphertext,
    encryptedBodyNonce: response.encryptedBodyNonce,
    encryptedPasteKeyCiphertext: response.encryptedPasteKeyCiphertext,
    encryptedPasteKeyNonce: response.encryptedPasteKeyNonce,
    pasteKeyBase64Url: response.pasteKeyBase64Url,
    pasteKey: response.pasteKey,
  };
}

export interface EncryptedFile {
  ciphertext: Uint8Array;
  nonce: Uint8Array;
}

export async function encryptFileInWorker(
  file: Uint8Array,
  key: Uint8Array,
  aad: Uint8Array,
): Promise<EncryptedFile> {
  const response = await postToWorker<EncryptFileResponse>(
    { type: "encryptFile", file, key, aad },
    [file.buffer],
  );
  return { ciphertext: response.ciphertext, nonce: response.nonce };
}

export async function decryptFileInWorker(
  ciphertext: Uint8Array,
  nonce: Uint8Array,
  key: Uint8Array,
  aad: Uint8Array,
): Promise<Uint8Array> {
  const response = await postToWorker<DecryptFileResponse>(
    { type: "decryptFile", ciphertext, nonce, key, aad },
    [ciphertext.buffer],
  );
  return response.plaintext;
}

function decryptString(ciphertextHex: string, nonceHex: string, key: Uint8Array, aad: Uint8Array): string {
  const ciphertext = hexToBytes(ciphertextHex);
  const nonce = hexToBytes(nonceHex);
  const plaintext = xchacha20poly1305(key, nonce, aad).decrypt(ciphertext);
  return new TextDecoder().decode(plaintext);
}

export function decryptPaste(
  vaultKey: Uint8Array,
  encryptedPasteKeyCiphertextHex: string,
  encryptedPasteKeyNonceHex: string,
  titleCiphertextHex: string,
  titleNonceHex: string,
  bodyCiphertextHex: string,
  bodyNonceHex: string,
): { title: string; body: string } {
  const encryptedKey = hexToBytes(encryptedPasteKeyCiphertextHex);
  const keyNonce = hexToBytes(encryptedPasteKeyNonceHex);
  const pasteKey = xchacha20poly1305(vaultKey, keyNonce, AAD.PASTE_KEY).decrypt(encryptedKey);
  const { contentKey } = derivePasteSubkeys(pasteKey);
  const title = decryptString(titleCiphertextHex, titleNonceHex, contentKey, AAD.TITLE);
  const body = decryptString(bodyCiphertextHex, bodyNonceHex, contentKey, AAD.BODY);
  return { title, body };
}

export function getPasteKeyFromHash(hashKeyBase64Url: string): Uint8Array {
  return base64UrlToBytes(hashKeyBase64Url);
}

export function getPasteKeyFromVault(
  vaultKey: Uint8Array,
  encryptedPasteKeyCiphertextHex: string,
  encryptedPasteKeyNonceHex: string,
): Uint8Array {
  const encryptedKey = hexToBytes(encryptedPasteKeyCiphertextHex);
  const keyNonce = hexToBytes(encryptedPasteKeyNonceHex);
  return xchacha20poly1305(vaultKey, keyNonce, AAD.PASTE_KEY).decrypt(encryptedKey);
}

export function decryptPasteWithKey(
  pasteKeyBase64Url: string,
  titleCiphertextHex: string,
  titleNonceHex: string,
  bodyCiphertextHex: string,
  bodyNonceHex: string,
): { title: string; body: string } {
  const pasteKey = base64UrlToBytes(pasteKeyBase64Url);
  const { contentKey } = derivePasteSubkeys(pasteKey);
  const title = decryptString(titleCiphertextHex, titleNonceHex, contentKey, AAD.TITLE);
  const body = decryptString(bodyCiphertextHex, bodyNonceHex, contentKey, AAD.BODY);
  return { title, body };
}

export function decryptPasteTitle(
  vaultKey: Uint8Array,
  encryptedPasteKeyCiphertextHex: string,
  encryptedPasteKeyNonceHex: string,
  titleCiphertextHex: string,
  titleNonceHex: string,
): string {
  const encryptedKey = hexToBytes(encryptedPasteKeyCiphertextHex);
  const keyNonce = hexToBytes(encryptedPasteKeyNonceHex);
  const pasteKey = xchacha20poly1305(vaultKey, keyNonce, AAD.PASTE_KEY).decrypt(encryptedKey);
  const { contentKey } = derivePasteSubkeys(pasteKey);
  return decryptString(titleCiphertextHex, titleNonceHex, contentKey, AAD.TITLE);
}

export function encryptFile(file: Uint8Array, key: Uint8Array, aad: Uint8Array): EncryptedFile {
  const nonce = randomBytes(24);
  const ciphertext = xchacha20poly1305(key, nonce, aad).encrypt(file);
  return { ciphertext, nonce };
}

export function decryptFile(ciphertext: Uint8Array, nonce: Uint8Array, key: Uint8Array, aad: Uint8Array): Uint8Array {
  return xchacha20poly1305(key, nonce, aad).decrypt(ciphertext);
}

export function encryptFilename(filename: string, key: Uint8Array, aad: Uint8Array): EncryptedFile {
  const content = encoder.encode(filename);
  return encryptFile(content, key, aad);
}

export function decryptFilename(ciphertextHex: string, nonceHex: string, key: Uint8Array, aad: Uint8Array): string {
  const ciphertext = hexToBytes(ciphertextHex);
  const nonce = hexToBytes(nonceHex);
  const plaintext = xchacha20poly1305(key, nonce, aad).decrypt(ciphertext);
  return new TextDecoder().decode(plaintext);
}
