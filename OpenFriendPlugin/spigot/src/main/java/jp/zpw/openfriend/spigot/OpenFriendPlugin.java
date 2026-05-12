/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package jp.zpw.openfriend.spigot;

import jp.zpw.openfriend.commons.BinaryRuntime;
import jp.zpw.openfriend.commons.Log;
import jp.zpw.openfriend.commons.PluginConfig;
import jp.zpw.openfriend.commons.ProcessSupervisor;
import jp.zpw.openfriend.commons.StatusSnapshot;
import jp.zpw.openfriend.commons.TaggedLogger;
import jp.zpw.openfriend.commons.VersionChecker;
import org.bukkit.ChatColor;
import org.bukkit.configuration.file.FileConfiguration;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.Listener;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.scheduler.BukkitRunnable;

import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;

public final class OpenFriendPlugin extends JavaPlugin implements Listener {

    private static final String BIN_DIR_NAME = "bin";
    private static final String MANIFEST_KEY = "OpenFriendPlugin";

    private ProcessSupervisor supervisor;
    private volatile VersionChecker.Result versionResult;
    private Log log;

    @Override
    public void onEnable() {
        saveDefaultConfig();
        log = TaggedLogger.of(getLogger(), "openfriend");
        getServer().getPluginManager().registerEvents(this, this);
        runVersionCheckAsync();

        PluginConfig cfg = readConfig();
        if (!cfg.enabled) {
            log.info("OpenFriend disabled in config.yml.");
            return;
        }
        if ("127.0.0.1:25565".equals(cfg.target)) {
            cfg.target = "127.0.0.1:" + getServer().getPort();
        }

        Path binDir = new File(getDataFolder(), BIN_DIR_NAME).toPath();
        try {
            Files.createDirectories(binDir);
        } catch (IOException e) {
            log.error("Failed to create bin dir: " + binDir, e);
            return;
        }

        String binaryName = BinaryRuntime.expectedBinaryName();
        Path binaryPath = binDir.resolve(binaryName);

        if (!Files.isExecutable(binaryPath)) {
            if (!BinaryRuntime.extractBundledBinary(OpenFriendPlugin.class, binaryName, binaryPath, log)) {
                log.error(BinaryRuntime.missingBinaryMessage(binaryPath));
                return;
            }
        }
        BinaryRuntime.makeExecutable(binaryPath);

        List<String> command = BinaryRuntime.buildCommand(binaryPath, cfg, getDataFolder().toPath());
        supervisor = new ProcessSupervisor(command, getDataFolder(), log, null);
        supervisor.start();
    }

    @Override
    public void onDisable() {
        if (supervisor != null) {
            supervisor.stop();
            supervisor = null;
        }
    }

    @EventHandler
    public void onJoin(PlayerJoinEvent event) {
        Player player = event.getPlayer();
        if (!player.isOp()) return;
        new BukkitRunnable() {
            @Override
            public void run() {
                sendOpStatus(player);
            }
        }.runTaskLater(this, 20L);
    }

    private void sendOpStatus(Player player) {
        StatusSnapshot s = StatusSnapshot.read(new File(getDataFolder(), "status.json").toPath());
        player.sendMessage(ChatColor.AQUA + "[OpenFriend] " + ChatColor.RESET + "status:");
        if (!s.authenticated) {
            player.sendMessage(ChatColor.YELLOW + "  not authenticated (waiting for device-code)");
            return;
        }
        player.sendMessage("  " + ChatColor.GRAY + "account: " + ChatColor.WHITE +
                (s.profileName.isEmpty() ? "?" : s.profileName) +
                ChatColor.GRAY + " (" + s.profileId + ")");
        player.sendMessage("  " + ChatColor.GRAY + "presence: " + ChatColor.WHITE +
                (s.presenceStatus.isEmpty() ? "unknown" : s.presenceStatus) +
                (s.presenceRunning ? ChatColor.GREEN + " ✓" : ChatColor.RED + " ✗"));
        player.sendMessage("  " + ChatColor.GRAY + "signaling: " +
                (s.signalingConnected ? ChatColor.GREEN + "connected" : ChatColor.RED + "disconnected"));
        player.sendMessage("  " + ChatColor.GRAY + "bypass: " +
                (s.bypassEnabled ? ChatColor.GREEN + "enabled" : ChatColor.YELLOW + "off (offline-mode only)"));
        VersionChecker.Result vr = versionResult;
        if (vr != null && vr.updateAvailable) {
            player.sendMessage(ChatColor.GOLD + "  update available: " + ChatColor.WHITE +
                    vr.current + " → " + vr.latest + ChatColor.GRAY + "  " + VersionChecker.DOWNLOAD_PAGE);
        }
    }

    private void runVersionCheckAsync() {
        new BukkitRunnable() {
            @Override
            public void run() {
                String current = getDescription().getVersion();
                versionResult = VersionChecker.check(MANIFEST_KEY, current);
                if (versionResult.updateAvailable) {
                    log.warn("Update available: " + versionResult.current + " -> " + versionResult.latest);
                    log.warn("Download: " + VersionChecker.DOWNLOAD_PAGE);
                }
            }
        }.runTaskAsynchronously(this);
    }

    private PluginConfig readConfig() {
        FileConfiguration yml = getConfig();
        PluginConfig cfg = PluginConfig.defaults();
        cfg.enabled = yml.getBoolean("enabled", cfg.enabled);
        cfg.target = yml.getString("target", cfg.target);
        cfg.intervalSeconds = yml.getInt("interval-s", cfg.intervalSeconds);
        cfg.noAutoAccept = yml.getBoolean("no-auto-accept", cfg.noAutoAccept);
        cfg.verbose = yml.getBoolean("verbose", cfg.verbose);
        cfg.skinFile = yml.getString("skin.file", cfg.skinFile);
        cfg.skinVariant = yml.getString("skin.variant", cfg.skinVariant);
        return cfg;
    }

}
