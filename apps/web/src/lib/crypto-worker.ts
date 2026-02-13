import { bytesToHex, hexToBytes, randomBytes } from "@noble/hashes/utils.js";
import { hmac } from "@noble/hashes/hmac.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { argon2id } from "@noble/hashes/argon2.js";
import { xchacha20poly1305 } from "@noble/ciphers/chacha.js";
import type {
  WorkerRequest,
  WorkerResponse,
  CryptoRequest,
  CryptoResponse,
} from "./crypto-worker-protocol";

const encoder = new TextEncoder();

function deriveSubkey(key: Uint8Array, context: string): Uint8Array {
  return hmac(sha256, key, encoder.encode(context));
}

const AAD = {
  TITLE: encoder.encode("skjul:v1:title"),
  BODY: encoder.encode("skjul:v1:body"),
  PASTE_KEY: encoder.encode("skjul:v1:paste_key"),
  FILE: encoder.encode("skjul:v1:file"),
  FILENAME: encoder.encode("skjul:v1:filename"),
  MIME: encoder.encode("skjul:v1:mime"),
  VAULT_KEY: encoder.encode("skjul:v1:vault_key"),
} as const;

function derivePasteSubkeys(pasteKey: Uint8Array) {
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

function encryptString(
  text: string,
  key: Uint8Array,
  aad: Uint8Array,
): { ciphertext: Uint8Array; nonce: Uint8Array } {
  const nonce = randomBytes(24);
  const content = encoder.encode(text);
  const ciphertext = xchacha20poly1305(key, nonce, aad).encrypt(content);
  return { ciphertext, nonce };
}

// Message handler

function handleRequest(req: CryptoRequest): {
  response: CryptoResponse;
  transfer: Transferable[];
} {
  switch (req.type) {
    case "generateRegistrationData": {
      const salt = randomBytes(16);
      const masterKey = argon2id(req.password, salt, {
        t: 2,
        m: 64 * 1024,
        p: 1,
        dkLen: 32,
      });
      const authKey = deriveSubkey(masterKey, "auth_v1");
      const encKey = deriveSubkey(masterKey, "master_enc_v1");
      const vaultKey = randomBytes(32);
      const vaultKeyNonce = randomBytes(24);
      const encryptedVaultKey = xchacha20poly1305(
        encKey,
        vaultKeyNonce,
        AAD.VAULT_KEY,
      ).encrypt(vaultKey);

      return {
        response: {
          type: "generateRegistrationData",
          salt: bytesToHex(salt),
          authKey: bytesToHex(authKey),
          encryptedVaultKey: bytesToHex(encryptedVaultKey),
          vaultKeyNonce: bytesToHex(vaultKeyNonce),
        },
        transfer: [],
      };
    }

    case "deriveLoginKeys": {
      const salt = hexToBytes(req.saltHex);
      const masterKey = argon2id(req.password, salt, {
        t: 2,
        m: 64 * 1024,
        p: 1,
        dkLen: 32,
      });
      const authKey = deriveSubkey(masterKey, "auth_v1");
      return {
        response: {
          type: "deriveLoginKeys",
          masterKey,
          authKey: bytesToHex(authKey),
        },
        transfer: [masterKey.buffer],
      };
    }

    case "decryptVaultKey": {
      const encKey = deriveSubkey(req.masterKey, "master_enc_v1");
      const encryptedVaultKey = hexToBytes(req.encryptedVaultKeyHex);
      const nonce = hexToBytes(req.nonceHex);
      const vaultKey = xchacha20poly1305(encKey, nonce, AAD.VAULT_KEY).decrypt(
        encryptedVaultKey,
      );
      return {
        response: { type: "decryptVaultKey", vaultKey },
        transfer: [vaultKey.buffer],
      };
    }

    case "createPaste": {
      const pasteKey = randomBytes(32);
      const { contentKey } = derivePasteSubkeys(pasteKey);
      const encryptedTitle = encryptString(req.title, contentKey, AAD.TITLE);
      const encryptedBody = encryptString(req.body, contentKey, AAD.BODY);
      const nonce = randomBytes(24);
      const encryptedKey = xchacha20poly1305(
        req.vaultKey,
        nonce,
        AAD.PASTE_KEY,
      ).encrypt(pasteKey);

      return {
        response: {
          type: "createPaste",
          encryptedTitleCiphertext: encryptedTitle.ciphertext,
          encryptedTitleNonce: encryptedTitle.nonce,
          encryptedBodyCiphertext: encryptedBody.ciphertext,
          encryptedBodyNonce: encryptedBody.nonce,
          encryptedPasteKeyCiphertext: encryptedKey,
          encryptedPasteKeyNonce: nonce,
          pasteKeyBase64Url: bytesToBase64Url(pasteKey),
          pasteKey,
        },
        transfer: [
          encryptedTitle.ciphertext.buffer,
          encryptedTitle.nonce.buffer,
          encryptedBody.ciphertext.buffer,
          encryptedBody.nonce.buffer,
          encryptedKey.buffer,
          nonce.buffer,
          pasteKey.buffer,
        ],
      };
    }

    case "encryptFile": {
      const nonce = randomBytes(24);
      const ciphertext = xchacha20poly1305(req.key, nonce, req.aad).encrypt(
        req.file,
      );
      return {
        response: { type: "encryptFile", ciphertext, nonce },
        transfer: [ciphertext.buffer, nonce.buffer],
      };
    }

    case "decryptFile": {
      const plaintext = xchacha20poly1305(
        req.key,
        req.nonce,
        req.aad,
      ).decrypt(req.ciphertext);
      return {
        response: { type: "decryptFile", plaintext },
        transfer: [plaintext.buffer],
      };
    }

    default: {
      throw new Error(`Unknown request type: ${(req as CryptoRequest).type}`);
    }
  }
}

self.onmessage = (event: MessageEvent<WorkerRequest>) => {
  const { id, request } = event.data;
  try {
    const { response, transfer } = handleRequest(request);
    const msg: WorkerResponse = { id, ok: true, response };
    (self as unknown as Worker).postMessage(msg, transfer);
  } catch (err) {
    const msg: WorkerResponse = {
      id,
      ok: false,
      error: err instanceof Error ? err.message : "Unknown worker error",
    };
    (self as unknown as Worker).postMessage(msg);
  }
};
