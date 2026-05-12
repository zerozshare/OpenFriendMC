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

public final class PluginConfig {
    public boolean enabled = true;
    public String target = "127.0.0.1:25565";
    public int intervalSeconds = 0;
    public boolean noAutoAccept = false;
    public boolean verbose = false;
    public String skinFile = "";
    public String skinVariant = "classic";

    public static PluginConfig defaults() {
        return new PluginConfig();
    }
}
