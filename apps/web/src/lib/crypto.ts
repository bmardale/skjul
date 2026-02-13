import { bytesToHex, hexToBytes, randomBytes } from "@noble/hashes/utils.js";
import { hmac } from "@noble/hashes/hmac.js"
import { sha256 } from "@noble/hashes/sha2.js"
import { argon2id } from "@noble/hashes/argon2.js";
import { xchacha20poly1305} from "@noble/ciphers/chacha.js"

function bytesToBase64Url(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

function base64UrlToBytes(str: string): Uint8Array {
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

function encryptString(text: string, key: Uint8Array) {
  const nonce = randomBytes(24);
  const content = new TextEncoder().encode(text);
  const ciphertext = xchacha20poly1305(key, nonce).encrypt(content);
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

    const authKey = hmac(sha256, masterKey, new TextEncoder().encode("auth_v1"));
    const vaultKey = randomBytes(32);
    const vaultKeyNonce = randomBytes(24);
    const encryptedVaultKey = xchacha20poly1305(masterKey, vaultKeyNonce).encrypt(vaultKey);

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
  const authKey = hmac(sha256, masterKey, new TextEncoder().encode("auth_v1"));
  return { masterKey, authKey: bytesToHex(authKey) };
}

export async function decryptVaultKey(
  masterKey: Uint8Array,
  encryptedVaultKeyHex: string,
  nonceHex: string,
): Promise<Uint8Array> {
  const encryptedVaultKey = hexToBytes(encryptedVaultKeyHex);
  const nonce = hexToBytes(nonceHex);
  return xchacha20poly1305(masterKey, nonce).decrypt(encryptedVaultKey);
}

interface PasteResult {
  encryptedTitleCiphertext: Uint8Array;
  encryptedTitleNonce: Uint8Array;
  encryptedBodyCiphertext: Uint8Array;
  encryptedBodyNonce: Uint8Array;
  encryptedPasteKeyCiphertext: Uint8Array;
  encryptedPasteKeyNonce: Uint8Array;
  pasteKeyBase64Url: string;
}

function decryptString(ciphertextHex: string, nonceHex: string, key: Uint8Array): string {
  const ciphertext = hexToBytes(ciphertextHex);
  const nonce = hexToBytes(nonceHex);
  const plaintext = xchacha20poly1305(key, nonce).decrypt(ciphertext);
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
  const pasteKey = xchacha20poly1305(vaultKey, keyNonce).decrypt(encryptedKey);
  const title = decryptString(titleCiphertextHex, titleNonceHex, pasteKey);
  const body = decryptString(bodyCiphertextHex, bodyNonceHex, pasteKey);
  return { title, body };
}

export function decryptPasteWithKey(
  pasteKeyBase64Url: string,
  titleCiphertextHex: string,
  titleNonceHex: string,
  bodyCiphertextHex: string,
  bodyNonceHex: string,
): { title: string; body: string } {
  const pasteKey = base64UrlToBytes(pasteKeyBase64Url);
  const title = decryptString(titleCiphertextHex, titleNonceHex, pasteKey);
  const body = decryptString(bodyCiphertextHex, bodyNonceHex, pasteKey);
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
  const pasteKey = xchacha20poly1305(vaultKey, keyNonce).decrypt(encryptedKey);
  return decryptString(titleCiphertextHex, titleNonceHex, pasteKey);
}

export async function createPaste(title: string, body: string, vaultKey: Uint8Array): Promise<PasteResult> {
  const pasteKey = randomBytes(32);
  const encryptedTitle = encryptString(title, pasteKey);
  const encryptedBody = encryptString(body, pasteKey);

  const nonce = randomBytes(24);
  const encryptedKey = xchacha20poly1305(vaultKey, nonce).encrypt(pasteKey);

  return {
    encryptedTitleCiphertext: encryptedTitle.ciphertext,
    encryptedTitleNonce: encryptedTitle.nonce,
    encryptedBodyCiphertext: encryptedBody.ciphertext,
    encryptedBodyNonce: encryptedBody.nonce,
    encryptedPasteKeyCiphertext: encryptedKey,
    encryptedPasteKeyNonce: nonce,
    pasteKeyBase64Url: bytesToBase64Url(pasteKey),
  }
  
  
}