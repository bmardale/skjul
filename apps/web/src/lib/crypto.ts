import { bytesToHex, hexToBytes, randomBytes } from "@noble/hashes/utils.js";
import { hmac } from "@noble/hashes/hmac.js"
import { sha256 } from "@noble/hashes/sha2.js"
import { argon2id } from "@noble/hashes/argon2.js";
import { xchacha20poly1305} from "@noble/ciphers/chacha.js"

interface RegistrationResult {
    salt: string;
    authKey: string;
    encryptedVaultKey: string;
    vaultKeyNonce: string;
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