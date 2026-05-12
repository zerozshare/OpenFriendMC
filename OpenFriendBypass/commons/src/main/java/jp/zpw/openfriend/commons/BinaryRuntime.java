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
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.nio.file.attribute.PosixFilePermission;
import java.util.ArrayList;
import java.util.EnumSet;
import java.util.List;
import java.util.Locale;
import java.util.Set;

public final class BinaryRuntime {

    public static final String DOWNLOAD_PAGE = "https://github.com/zerozshare/OpenFriendMC/releases";
    public static final String BUNDLED_RESOURCE_PREFIX = "/openfriend/bin/";

    private BinaryRuntime() {
    }

    public static String expectedBinaryName() {
        String os = System.getProperty("os.name", "").toLowerCase(Locale.ROOT);
        String arch = System.getProperty("os.arch", "").toLowerCase(Locale.ROOT);
        String osTag;
        String archTag;
        String ext = "";
        if (os.contains("mac") || os.contains("darwin")) {
            osTag = "darwin";
        } else if (os.contains("linux")) {
            osTag = "linux";
        } else if (os.contains("win")) {
            osTag = "windows";
            ext = ".exe";
        } else {
            osTag = "unknown";
        }
        if (arch.contains("aarch64") || arch.contains("arm64")) {
            archTag = "arm64";
        } else if (arch.contains("64")) {
            archTag = "amd64";
        } else {
            archTag = "unknown";
        }
        return "openfriend-" + osTag + "-" + archTag + ext;
    }

    public static boolean extractBundledBinary(Class<?> caller, String resourceName, Path destination, Log log) {
        String resourcePath = BUNDLED_RESOURCE_PREFIX + resourceName;
        InputStream in = caller.getResourceAsStream(resourcePath);
        if (in == null) {
            log.warn("Bundled binary not present at resource: " + resourcePath);
            return false;
        }
        try {
            try {
                Files.copy(in, destination, StandardCopyOption.REPLACE_EXISTING);
            } finally {
                in.close();
            }
            log.info("Extracted bundled binary to " + destination.toAbsolutePath());
            return true;
        } catch (IOException e) {
            log.warn("Failed to extract bundled binary", e);
            return false;
        }
    }

    public static void makeExecutable(Path path) {
        Set<PosixFilePermission> perms = EnumSet.of(
                PosixFilePermission.OWNER_READ,
                PosixFilePermission.OWNER_WRITE,
                PosixFilePermission.OWNER_EXECUTE,
                PosixFilePermission.GROUP_READ,
                PosixFilePermission.GROUP_EXECUTE,
                PosixFilePermission.OTHERS_READ,
                PosixFilePermission.OTHERS_EXECUTE);
        try {
            Files.setPosixFilePermissions(path, perms);
        } catch (UnsupportedOperationException ignored) {
        } catch (IOException ignored) {
        }
    }

    public static List<String> buildCommand(Path binaryPath, PluginConfig cfg, Path dataDir) {
        List<String> args = new ArrayList<String>();
        args.add(binaryPath.toAbsolutePath().toString());
        args.add("--watch-parent");
        args.add("--data-dir");
        args.add(dataDir.toAbsolutePath().toString());
        args.add("--target");
        args.add(cfg.target);
        if (cfg.intervalSeconds > 0) {
            args.add("--interval-s");
            args.add(String.valueOf(cfg.intervalSeconds));
        }
        if (cfg.noAutoAccept) {
            args.add("--no-auto-accept");
        }
        if (cfg.verbose) {
            args.add("--verbose");
        }
        if (cfg.skinFile != null && !cfg.skinFile.isEmpty()) {
            Path skinPath = dataDir.resolve(cfg.skinFile).toAbsolutePath();
            args.add("--skin");
            args.add(skinPath.toString());
            args.add("--skin-variant");
            args.add(cfg.skinVariant == null || cfg.skinVariant.isEmpty() ? "classic" : cfg.skinVariant);
        }
        args.add("--no-update");
        return args;
    }

    public static String missingBinaryMessage(Path expectedPath) {
        return "OpenFriend binary not found. Download for your platform from "
                + DOWNLOAD_PAGE
                + " and place it at: " + expectedPath.toAbsolutePath();
    }
}
