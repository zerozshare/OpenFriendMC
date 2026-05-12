/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package jp.zpw.openfriend.bypass;

import io.netty.channel.Channel;
import io.netty.util.AttributeKey;
import jp.zpw.openfriend.commons.Log;

import java.lang.reflect.Field;
import java.lang.reflect.Method;
import java.util.Locale;

public final class EncryptionBypass {

    public static final AttributeKey<Boolean> TRUSTED = AttributeKey.valueOf("openfriend.bypass.trusted");

    private static final String[] VERIFYING_STATE_NAMES = {"VERIFYING", "AUTHENTICATING", "READY_TO_ACCEPT", "ACCEPTED"};

    private static Class<?> connectionClass;
    private static Class<?> loginListenerClass;
    private static Object stateValueAfterAuth;
    private static Field stateField;
    private static Field authProfileField;
    private static Field requestedUsernameField;
    private static Field packetListenerField;
    private static Method createOfflineProfileMethod;
    private static volatile boolean ready;

    private EncryptionBypass() {
    }

    public static void install(Log log) {
        try {
            connectionClass = Class.forName("net.minecraft.network.Connection");
            loginListenerClass = Class.forName("net.minecraft.server.network.ServerLoginPacketListenerImpl");
            stateField = findEnumField(loginListenerClass, "state");
            if (stateField == null) {
                log.warn("Could not locate enum 'state' field on ServerLoginPacketListenerImpl");
                return;
            }
            stateField.setAccessible(true);
            stateValueAfterAuth = pickPostAuthStateValue(stateField.getType());
            if (stateValueAfterAuth == null) {
                log.warn("Could not locate a usable post-auth state in " + stateField.getType().getName());
                return;
            }
            authProfileField = findFieldByTypeNameSuffix(loginListenerClass, "GameProfile");
            if (authProfileField == null) {
                log.warn("Could not locate GameProfile field on ServerLoginPacketListenerImpl");
                return;
            }
            authProfileField.setAccessible(true);
            requestedUsernameField = findStringFieldByNameHint(loginListenerClass, "name", "username", "requested");
            if (requestedUsernameField == null) {
                log.warn("Could not locate requested username field on ServerLoginPacketListenerImpl");
                return;
            }
            requestedUsernameField.setAccessible(true);
            packetListenerField = findFieldByInterfaceContains(connectionClass, "PacketListener");
            if (packetListenerField == null) {
                log.warn("Could not locate packetListener field on Connection");
                return;
            }
            packetListenerField.setAccessible(true);
            createOfflineProfileMethod = resolveCreateOfflineProfile();
            if (createOfflineProfileMethod == null) {
                log.warn("Could not locate UUIDUtil.createOfflineProfile");
                return;
            }
            ready = true;
            log.info("NMS reflection ready (state=" + stateValueAfterAuth + ", profileField=" + authProfileField.getName() +
                    ", usernameField=" + requestedUsernameField.getName() + ")");
        } catch (Throwable t) {
            log.warn("Failed to install bypass reflection on this server version", t);
        }
    }

    private static Field findEnumField(Class<?> owner, String nameHint) {
        Field best = null;
        for (Field f : owner.getDeclaredFields()) {
            if (!f.getType().isEnum()) continue;
            if (f.getName().toLowerCase(Locale.ROOT).contains(nameHint)) {
                return f;
            }
            if (best == null) best = f;
        }
        return best;
    }

    private static Object pickPostAuthStateValue(Class<?> enumClass) {
        Object[] constants = enumClass.getEnumConstants();
        for (String want : VERIFYING_STATE_NAMES) {
            for (Object c : constants) {
                if (((Enum<?>) c).name().equals(want)) return c;
            }
        }
        return null;
    }

    private static Field findFieldByTypeNameSuffix(Class<?> owner, String typeSuffix) {
        for (Field f : owner.getDeclaredFields()) {
            String typeName = f.getType().getName();
            if (typeName.endsWith("." + typeSuffix) || typeName.endsWith("$" + typeSuffix) || typeName.equals(typeSuffix)) {
                return f;
            }
        }
        return null;
    }

    private static Field findStringFieldByNameHint(Class<?> owner, String... hints) {
        for (Field f : owner.getDeclaredFields()) {
            if (!f.getType().equals(String.class)) continue;
            String n = f.getName().toLowerCase(Locale.ROOT);
            for (String hint : hints) {
                if (n.contains(hint)) return f;
            }
        }
        for (Field f : owner.getDeclaredFields()) {
            if (f.getType().equals(String.class)) return f;
        }
        return null;
    }

    private static Field findFieldByInterfaceContains(Class<?> owner, String typeNameContains) {
        for (Field f : owner.getDeclaredFields()) {
            if (f.getType().getName().contains(typeNameContains)) return f;
        }
        return null;
    }

    private static Method resolveCreateOfflineProfile() {
        String[] classNames = {
                "net.minecraft.core.UUIDUtil",
                "net.minecraft.util.UUIDUtil",
        };
        for (String cn : classNames) {
            try {
                Class<?> c = Class.forName(cn);
                for (Method m : c.getMethods()) {
                    if (!m.getName().toLowerCase(Locale.ROOT).contains("offline")) continue;
                    if (m.getParameterCount() != 1) continue;
                    if (!m.getParameterTypes()[0].equals(String.class)) continue;
                    return m;
                }
            } catch (ClassNotFoundException ignored) {
            }
        }
        return null;
    }

    public static boolean isReady() {
        return ready;
    }

    public static void markTrusted(Channel channel) {
        if (channel != null) {
            channel.attr(TRUSTED).set(Boolean.TRUE);
        }
    }

    public static boolean isTrusted(Channel channel) {
        if (channel == null) return false;
        Boolean v = channel.attr(TRUSTED).get();
        return v != null && v;
    }

    public static boolean tryBypassLogin(Channel channel, String playerName, Log log) {
        if (!ready) return false;
        if (!isTrusted(channel)) return false;
        try {
            Object connection = locateConnection(channel);
            if (connection == null) {
                log.warn("could not find Connection handler on channel");
                return false;
            }
            Object listener = packetListenerField.get(connection);
            if (listener == null || !loginListenerClass.isInstance(listener)) {
                log.warn("connection's packetListener is not a ServerLoginPacketListenerImpl");
                return false;
            }
            Object profile = createOfflineProfileMethod.invoke(null, playerName);
            requestedUsernameField.set(listener, playerName);
            authProfileField.set(listener, profile);
            stateField.set(listener, stateValueAfterAuth);
            log.info("login state forced to " + stateValueAfterAuth + " for " + playerName);
            return true;
        } catch (Throwable t) {
            log.warn("failed to force login state", t);
            return false;
        }
    }

    private static Object locateConnection(Channel channel) {
        if (channel == null) return null;
        Object handler = channel.pipeline().get("packet_handler");
        if (handler != null && connectionClass.isInstance(handler)) {
            return handler;
        }
        for (String name : channel.pipeline().names()) {
            Object h = channel.pipeline().get(name);
            if (h != null && connectionClass.isInstance(h)) {
                return h;
            }
        }
        return null;
    }
}
