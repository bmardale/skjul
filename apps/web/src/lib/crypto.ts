import { bytesToHex, hexToBytes, randomBytes } from "@noble/hashes/utils.js";
import { hmac } from "@noble/hashes/hmac.js"
import { sha256 } from "@noble/hashes/sha2.js"
import { argon2id } from "@noble/hashes/argon2.js";
import { xchacha20poly1305} from "@noble/ciphers/chacha.js"

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

function bytesToBase64Url(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

export function base64UrlToBytes(str: string): Uint8Array {
  let base64 = str.replace(/-/g, "+").replace(/_/g, "/");
  while (base64.length % 4) base64 += "=";
  const binary = atob(base64);
  return Uint8Array.from(binary, (c) => c.charCodeAt(0));
}

interface RegistrationResult {
    salt: string;
    authKey: string;
    encryptedVaultKey: string;
    vaultKeyNonce: string;
}

function encryptString(text: string, key: Uint8Array, aad: Uint8Array) {
  const nonce = randomBytes(24);
  const content = encoder.encode(text);
  const ciphertext = xchacha20poly1305(key, nonce, aad).encrypt(content);
  return { ciphertext, nonce };
}

export async function generateRegistrationData(password: string): Promise<RegistrationResult> {
    const salt = randomBytes(16);
    const masterKey = argon2id(password, salt, {
        t: 2,
        m:  64 * 1024,
        p: 1,
        dkLen: 32,
    });

    const authKey = deriveSubkey(masterKey, "auth_v1");
    const encKey = deriveSubkey(masterKey, "master_enc_v1");
    const vaultKey = randomBytes(32);
    const vaultKeyNonce = randomBytes(24);
    const encryptedVaultKey = xchacha20poly1305(encKey, vaultKeyNonce, AAD.VAULT_KEY).encrypt(vaultKey);

    return {
        salt: bytesToHex(salt),
        authKey: bytesToHex(authKey),
        encryptedVaultKey: bytesToHex(encryptedVaultKey),
        vaultKeyNonce: bytesToHex(vaultKeyNonce),
    }
}

export async function deriveLoginKeys(
  password: string,
  saltHex: string,
): Promise<{ masterKey: Uint8Array; authKey: string }> {
  const salt = hexToBytes(saltHex);
  const masterKey = argon2id(password, salt, {
    t: 2,
    m: 64 * 1024,
    p: 1,
    dkLen: 32,
  });
  const authKey = deriveSubkey(masterKey, "auth_v1");
  return { masterKey, authKey: bytesToHex(authKey) };
}

export async function decryptVaultKey(
  masterKey: Uint8Array,
  encryptedVaultKeyHex: string,
  nonceHex: string,
): Promise<Uint8Array> {
  const encKey = deriveSubkey(masterKey, "master_enc_v1");
  const encryptedVaultKey = hexToBytes(encryptedVaultKeyHex);
  const nonce = hexToBytes(nonceHex);
  return xchacha20poly1305(encKey, nonce, AAD.VAULT_KEY).decrypt(encryptedVaultKey);
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

export async function createPaste(title: string, body: string, vaultKey: Uint8Array): Promise<PasteResult> {
  const pasteKey = randomBytes(32);
  const { contentKey } = derivePasteSubkeys(pasteKey);
  const encryptedTitle = encryptString(title, contentKey, AAD.TITLE);
  const encryptedBody = encryptString(body, contentKey, AAD.BODY);

  const nonce = randomBytes(24);
  const encryptedKey = xchacha20poly1305(vaultKey, nonce, AAD.PASTE_KEY).encrypt(pasteKey);

  return {
    encryptedTitleCiphertext: encryptedTitle.ciphertext,
    encryptedTitleNonce: encryptedTitle.nonce,
    encryptedBodyCiphertext: encryptedBody.ciphertext,
    encryptedBodyNonce: encryptedBody.nonce,
    encryptedPasteKeyCiphertext: encryptedKey,
    encryptedPasteKeyNonce: nonce,
    pasteKeyBase64Url: bytesToBase64Url(pasteKey),
    pasteKey,
  }
}

export interface EncryptedFile {
  ciphertext: Uint8Array;
  nonce: Uint8Array;
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