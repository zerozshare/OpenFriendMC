/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package jp.zpw.openfriend.commons;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.SecureRandom;
import java.util.Base64;

public final class BypassKey {

    public static final String PEM_TYPE = "OPENFRIEND BYPASS KEY";
    public static final int KEY_BYTES = 32;
    public static final String MARKER_SEGMENT = "openfriend";
    private static final String HEADER_TEXT =
            "# OpenFriend bypass shared secret. Treat this file like a password.\n" +
            "# The same file must be present on the Minecraft server (next to OpenFriendBypass)\n" +
            "# and on the OpenFriend Go bridge (--bypass-key or in --data-dir).\n";

    private final byte[] bytes;

    private BypassKey(byte[] bytes) {
        this.bytes = bytes;
    }

    public byte[] bytes() {
        return bytes;
    }

    public static BypassKey load(Path path) throws IOException {
        byte[] raw = Files.readAllBytes(path);
        String pem = new String(raw, StandardCharsets.UTF_8);
        int begin = pem.indexOf("-----BEGIN " + PEM_TYPE + "-----");
        int end = pem.indexOf("-----END " + PEM_TYPE + "-----");
        if (begin < 0 || end < 0) {
            throw new IOException("not a " + PEM_TYPE + " PEM file");
        }
        String body = pem.substring(begin + ("-----BEGIN " + PEM_TYPE + "-----").length(), end);
        body = body.replaceAll("\\s", "");
        byte[] key = Base64.getDecoder().decode(body);
        if (key.length != KEY_BYTES) {
            throw new IOException("bypass key must be " + KEY_BYTES + " bytes, got " + key.length);
        }
        return new BypassKey(key);
    }

    public static BypassKey generate(Path path) throws IOException {
        byte[] key = new byte[KEY_BYTES];
        new SecureRandom().nextBytes(key);
        if (path.getParent() != null) {
            Files.createDirectories(path.getParent());
        }
        StringBuilder sb = new StringBuilder();
        sb.append(HEADER_TEXT);
        sb.append("-----BEGIN ").append(PEM_TYPE).append("-----\n");
        String encoded = Base64.getEncoder().encodeToString(key);
        for (int i = 0; i < encoded.length(); i += 64) {
            sb.append(encoded, i, Math.min(i + 64, encoded.length())).append('\n');
        }
        sb.append("-----END ").append(PEM_TYPE).append("-----\n");
        Files.write(path, sb.toString().getBytes(StandardCharsets.UTF_8));
        return new BypassKey(key);
    }

    public static BypassKey loadOrCreate(Path path) throws IOException {
        if (Files.isRegularFile(path)) {
            return load(path);
        }
        return generate(path);
    }

    public boolean verify(String nonceB64, String signatureB64) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(bytes, "HmacSHA256"));
            byte[] expected = mac.doFinal(nonceB64.getBytes(StandardCharsets.UTF_8));
            byte[] actual;
            try {
                actual = Base64.getDecoder().decode(signatureB64);
            } catch (IllegalArgumentException e) {
                return false;
            }
            return constantTimeEquals(expected, actual);
        } catch (Exception e) {
            return false;
        }
    }

    private static boolean constantTimeEquals(byte[] a, byte[] b) {
        if (a.length != b.length) return false;
        int diff = 0;
        for (int i = 0; i < a.length; i++) diff |= a[i] ^ b[i];
        return diff == 0;
    }

    public static final class MarkerData {
        public final String originalAddress;
        public final String nonce;
        public final String signature;

        public MarkerData(String originalAddress, String nonce, String signature) {
            this.originalAddress = originalAddress;
            this.nonce = nonce;
            this.signature = signature;
        }
    }

    public static MarkerData extract(String addressField) {
        int idx = addressField.indexOf("\0" + MARKER_SEGMENT + "\0");
        if (idx < 0) return null;
        String original = addressField.substring(0, idx);
        String rest = addressField.substring(idx + MARKER_SEGMENT.length() + 2);
        String[] parts = rest.split("\0");
        if (parts.length != 2) return null;
        return new MarkerData(original, parts[0], parts[1]);
    }
}
