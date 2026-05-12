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

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;

public final class StatusSnapshot {

    public boolean authenticated;
    public String profileId = "";
    public String profileName = "";
    public String presenceStatus = "";
    public boolean presenceRunning;
    public boolean signalingConnected;
    public boolean bypassEnabled;
    public String version = "";
    public String updatedAt = "";

    public static StatusSnapshot read(Path file) {
        StatusSnapshot s = new StatusSnapshot();
        if (!Files.isRegularFile(file)) return s;
        try {
            String json = new String(Files.readAllBytes(file), StandardCharsets.UTF_8);
            s.authenticated = parseBool(json, "authenticated");
            s.profileId = parseString(json, "profile_id");
            s.profileName = parseString(json, "profile_name");
            s.presenceStatus = parseString(json, "presence_status");
            s.presenceRunning = parseBool(json, "presence_running");
            s.signalingConnected = parseBool(json, "signaling_connected");
            s.bypassEnabled = parseBool(json, "bypass_enabled");
            s.version = parseString(json, "version");
            s.updatedAt = parseString(json, "updated_at");
        } catch (IOException ignored) {
        }
        return s;
    }

    private static String parseString(String json, String key) {
        String needle = "\"" + key + "\"";
        int idx = json.indexOf(needle);
        if (idx < 0) return "";
        int colon = json.indexOf(':', idx + needle.length());
        if (colon < 0) return "";
        int start = json.indexOf('"', colon + 1);
        if (start < 0) return "";
        int end = json.indexOf('"', start + 1);
        if (end < 0) return "";
        return json.substring(start + 1, end);
    }

    private static boolean parseBool(String json, String key) {
        String needle = "\"" + key + "\"";
        int idx = json.indexOf(needle);
        if (idx < 0) return false;
        int colon = json.indexOf(':', idx + needle.length());
        if (colon < 0) return false;
        for (int i = colon + 1; i < json.length(); i++) {
            char c = json.charAt(i);
            if (c == ' ' || c == '\t' || c == '\n' || c == '\r') continue;
            return json.startsWith("true", i);
        }
        return false;
    }
}
