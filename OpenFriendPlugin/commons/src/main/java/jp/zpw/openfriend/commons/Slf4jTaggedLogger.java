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

import org.slf4j.Logger;

public final class Slf4jTaggedLogger implements Log {

    private final Logger underlying;
    private final String prefix;

    public Slf4jTaggedLogger(Logger underlying, String tag) {
        this.underlying = underlying;
        this.prefix = "[" + tag + "] ";
    }

    @Override
    public void info(String message) {
        underlying.info(prefix + message);
    }

    @Override
    public void warn(String message) {
        underlying.warn(prefix + message);
    }

    @Override
    public void warn(String message, Throwable t) {
        underlying.warn(prefix + message, t);
    }

    @Override
    public void error(String message) {
        underlying.error(prefix + message);
    }

    @Override
    public void error(String message, Throwable t) {
        underlying.error(prefix + message, t);
    }
}
