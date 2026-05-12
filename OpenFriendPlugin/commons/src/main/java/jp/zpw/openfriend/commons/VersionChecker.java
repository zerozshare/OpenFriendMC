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

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;

public final class VersionChecker {

    public static final String DEFAULT_MANIFEST_URL = "https://api.zpw.jp/version/";
    public static final String DOWNLOAD_PAGE = "https://openfriend.net/";

    public static final class Result {
        public final boolean updateAvailable;
        public final String latest;
        public final String current;

        public Result(boolean updateAvailable, String latest, String current) {
            this.updateAvailable = updateAvailable;
            this.latest = latest;
            this.current = current;
        }
    }

    private VersionChecker() {
    }

    public static Result check(String manifestKey, String currentVersion) {
        try {
            String raw = fetch(DEFAULT_MANIFEST_URL);
            String latest = extractKey(raw, manifestKey);
            if (latest == null || latest.isEmpty()) {
                return new Result(false, latest, currentVersion);
            }
            boolean newer = compareVersions(latest, currentVersion) > 0;
            return new Result(newer, latest, currentVersion);
        } catch (Throwable t) {
            return new Result(false, null, currentVersion);
        }
    }

    private static String fetch(String url) throws Exception {
        HttpURLConnection conn = (HttpURLConnection) new URL(url).openConnection();
        try {
            conn.setConnectTimeout(5000);
            conn.setReadTimeout(5000);
            conn.setRequestMethod("GET");
            if (conn.getResponseCode() != 200) {
                throw new RuntimeException("status " + conn.getResponseCode());
            }
            StringBuilder sb = new StringBuilder();
            try (BufferedReader r = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8))) {
                String line;
                while ((line = r.readLine()) != null) sb.append(line);
            }
            return sb.toString();
        } finally {
            conn.disconnect();
        }
    }

    private static String extractKey(String json, String key) {
        String needle = "\"" + key + "\"";
        int idx = json.indexOf(needle);
        if (idx < 0) return null;
        int colon = json.indexOf(':', idx + needle.length());
        if (colon < 0) return null;
        int start = json.indexOf('"', colon + 1);
        if (start < 0) return null;
        int end = json.indexOf('"', start + 1);
        if (end < 0) return null;
        return json.substring(start + 1, end);
    }

    public static int compareVersions(String a, String b) {
        if (a == null || b == null) return 0;
        if (b.equalsIgnoreCase("dev")) return 1;
        if (a.equalsIgnoreCase("dev")) return -1;
        String[] aa = strip(a).split("\\.");
        String[] bb = strip(b).split("\\.");
        int max = Math.max(aa.length, bb.length);
        for (int i = 0; i < max; i++) {
            int x = i < aa.length ? parseSafe(aa[i]) : 0;
            int y = i < bb.length ? parseSafe(bb[i]) : 0;
            if (x != y) return Integer.compare(x, y);
        }
        return 0;
    }

    private static String strip(String s) {
        s = s.trim();
        if (s.startsWith("v") || s.startsWith("V")) s = s.substring(1);
        return s;
    }

    private static int parseSafe(String s) {
        StringBuilder digits = new StringBuilder();
        for (char c : s.toCharArray()) {
            if (c >= '0' && c <= '9') digits.append(c);
            else break;
        }
        if (digits.length() == 0) return 0;
        try {
            return Integer.parseInt(digits.toString());
        } catch (NumberFormatException e) {
            return 0;
        }
    }
}
