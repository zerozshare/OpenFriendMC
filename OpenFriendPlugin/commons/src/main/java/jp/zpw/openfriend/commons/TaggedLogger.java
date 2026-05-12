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

import java.util.logging.Level;
import java.util.logging.Logger;

public final class TaggedLogger implements Log {

    public static final String PREFIX_OPEN = "[";
    public static final String PREFIX_CLOSE = "] ";

    private final Logger underlying;
    private final String prefix;

    private TaggedLogger(Logger underlying, String tag) {
        this.underlying = underlying;
        this.prefix = PREFIX_OPEN + tag + PREFIX_CLOSE;
    }

    public static TaggedLogger of(Logger underlying, String tag) {
        return new TaggedLogger(underlying, tag);
    }

    public TaggedLogger sub(String tag) {
        return new TaggedLogger(underlying, raw() + " " + tag);
    }

    private String raw() {
        if (prefix.length() < PREFIX_OPEN.length() + PREFIX_CLOSE.length()) return "";
        return prefix.substring(PREFIX_OPEN.length(), prefix.length() - PREFIX_CLOSE.length());
    }

    @Override
    public void info(String message) {
        underlying.log(Level.INFO, prefix + message);
    }

    @Override
    public void warn(String message) {
        underlying.log(Level.WARNING, prefix + message);
    }

    @Override
    public void warn(String message, Throwable t) {
        underlying.log(Level.WARNING, prefix + message, t);
    }

    @Override
    public void error(String message) {
        underlying.log(Level.SEVERE, prefix + message);
    }

    @Override
    public void error(String message, Throwable t) {
        underlying.log(Level.SEVERE, prefix + message, t);
    }
}
