/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package jp.zpw.openfriend.velocity;

import com.google.inject.Inject;
import com.velocitypowered.api.event.Subscribe;
import com.velocitypowered.api.event.proxy.ProxyInitializeEvent;
import com.velocitypowered.api.event.proxy.ProxyShutdownEvent;
import com.velocitypowered.api.plugin.Plugin;
import com.velocitypowered.api.plugin.annotation.DataDirectory;
import com.velocitypowered.api.proxy.ProxyServer;
import jp.zpw.openfriend.commons.BinaryRuntime;
import jp.zpw.openfriend.commons.Log;
import jp.zpw.openfriend.commons.PluginConfig;
import jp.zpw.openfriend.commons.ProcessSupervisor;
import jp.zpw.openfriend.commons.Slf4jTaggedLogger;
import org.slf4j.Logger;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Properties;

@Plugin(id = "openfriend", name = "OpenFriend", version = "0.1.0", authors = {"ZSHARE"},
        description = "Bridge Minecraft Java Friends List joins (snapshot 26.2+) to a backend server.",
        url = "https://github.com/zerozshare/OpenFriendMC")
public final class OpenFriendVelocityPlugin {

    @SuppressWarnings("unused")
    private final ProxyServer server;
    @SuppressWarnings("unused")
    private final Logger logger;
    private final Path dataDir;
    private final Log log;

    private ProcessSupervisor supervisor;

    @Inject
    public OpenFriendVelocityPlugin(ProxyServer server, Logger logger, @DataDirectory Path dataDir) {
        this.server = server;
        this.logger = logger;
        this.dataDir = dataDir;
        this.log = new Slf4jTaggedLogger(logger, "openfriend");
    }

    @Subscribe
    public void onInit(ProxyInitializeEvent event) {
        try {
            Files.createDirectories(dataDir);
        } catch (IOException e) {
            log.error("Failed to create data dir: " + dataDir, e);
            return;
        }

        PluginConfig cfg = loadOrCreateConfig();
        if (!cfg.enabled) {
            log.info("OpenFriend disabled in config.");
            return;
        }

        Path binDir = dataDir.resolve("bin");
        try {
            Files.createDirectories(binDir);
        } catch (IOException e) {
            log.error("Failed to create bin dir: " + binDir, e);
            return;
        }

        String binaryName = BinaryRuntime.expectedBinaryName();
        Path binaryPath = binDir.resolve(binaryName);

        if (!Files.isExecutable(binaryPath)) {
            if (!BinaryRuntime.extractBundledBinary(OpenFriendVelocityPlugin.class, binaryName, binaryPath, log)) {
                log.error(BinaryRuntime.missingBinaryMessage(binaryPath));
                return;
            }
        }
        BinaryRuntime.makeExecutable(binaryPath);

        List<String> command = BinaryRuntime.buildCommand(binaryPath, cfg, dataDir);
        supervisor = new ProcessSupervisor(command, dataDir.toFile(), log, null);
        supervisor.start();
    }

    @Subscribe
    public void onShutdown(ProxyShutdownEvent event) {
        if (supervisor != null) {
            supervisor.stop();
            supervisor = null;
        }
    }

    private PluginConfig loadOrCreateConfig() {
        Path cfgPath = dataDir.resolve("config.properties");
        PluginConfig cfg = PluginConfig.defaults();
        Properties p = new Properties();
        if (Files.isRegularFile(cfgPath)) {
            try (InputStream in = Files.newInputStream(cfgPath)) {
                p.load(in);
            } catch (IOException e) {
                log.warn("Could not read config", e);
            }
            cfg.enabled = Boolean.parseBoolean(p.getProperty("enabled", String.valueOf(cfg.enabled)));
            cfg.target = p.getProperty("target", cfg.target);
            cfg.intervalSeconds = parseInt(p.getProperty("interval-s"), cfg.intervalSeconds);
            cfg.noAutoAccept = Boolean.parseBoolean(p.getProperty("no-auto-accept", String.valueOf(cfg.noAutoAccept)));
            cfg.verbose = Boolean.parseBoolean(p.getProperty("verbose", String.valueOf(cfg.verbose)));
            cfg.skinFile = p.getProperty("skin", cfg.skinFile);
            cfg.skinVariant = p.getProperty("skin-variant", cfg.skinVariant);
        } else {
            p.setProperty("enabled", String.valueOf(cfg.enabled));
            p.setProperty("target", cfg.target);
            p.setProperty("interval-s", String.valueOf(cfg.intervalSeconds == 0 ? 30 : cfg.intervalSeconds));
            try (OutputStream out = Files.newOutputStream(cfgPath)) {
                p.store(out, "OpenFriend Velocity config");
            } catch (IOException e) {
                log.warn("Could not write default config", e);
            }
        }
        return cfg;
    }

    private static int parseInt(String s, int fallback) {
        if (s == null || s.isEmpty()) return fallback;
        try {
            return Integer.parseInt(s.trim());
        } catch (NumberFormatException e) {
            return fallback;
        }
    }

}
