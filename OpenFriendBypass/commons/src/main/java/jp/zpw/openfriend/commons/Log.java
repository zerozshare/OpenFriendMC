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

public interface Log {
    void info(String message);
    void warn(String message);
    void warn(String message, Throwable t);
    void error(String message);
    void error(String message, Throwable t);
}
