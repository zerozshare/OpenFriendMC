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

import com.github.retrooper.packetevents.PacketEvents;
import com.github.retrooper.packetevents.event.PacketListenerAbstract;
import com.github.retrooper.packetevents.event.PacketReceiveEvent;
import com.github.retrooper.packetevents.protocol.packettype.PacketType;
import com.github.retrooper.packetevents.wrapper.handshaking.client.WrapperHandshakingClientHandshake;
import com.github.retrooper.packetevents.wrapper.login.client.WrapperLoginClientLoginStart;
import io.github.retrooper.packetevents.factory.spigot.SpigotPacketEventsBuilder;
import io.netty.channel.Channel;
import jp.zpw.openfriend.commons.BypassKey;
import jp.zpw.openfriend.commons.Log;
import jp.zpw.openfriend.commons.TaggedLogger;
import jp.zpw.openfriend.commons.VersionChecker;
import org.bukkit.ChatColor;
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
import java.nio.file.StandardCopyOption;
import java.util.concurrent.atomic.AtomicInteger;

public final class BypassPlugin extends JavaPlugin implements Listener {

    private static final String MANIFEST_KEY = "OpenFriendBypass";

    private BypassKey key;
    private Log log;
    private volatile VersionChecker.Result versionResult;
    private final AtomicInteger bypassedCount = new AtomicInteger();

    @Override
    public void onLoad() {
        log = TaggedLogger.of(getLogger(), "bypass");
        try {
            PacketEvents.setAPI(SpigotPacketEventsBuilder.build(this));
            PacketEvents.getAPI().load();
        } catch (Throwable t) {
            log.error("Failed to initialize packetevents", t);
        }
    }

    @Override
    public void onEnable() {
        Path keyPath = new File(getDataFolder(), "bypass.pem").toPath();
        boolean newlyGenerated = !Files.isRegularFile(keyPath);
        try {
            key = BypassKey.loadOrCreate(keyPath);
            log.info("Loaded bypass key from " + keyPath.toAbsolutePath());
        } catch (IOException e) {
            log.error("Failed to load/create bypass key; bypass is disabled", e);
            return;
        }

        boolean copied = syncKeyToBridgePlugin(keyPath);
        if (newlyGenerated && copied) {
            log.warn("==========================================================");
            log.warn("bypass.pem was generated and copied to plugins/OpenFriend/.");
            log.warn("Please RESTART the server so the OpenFriend Go bridge");
            log.warn("picks up the new key and the online-mode bypass becomes active.");
            log.warn("==========================================================");
        }

        PacketEvents.getAPI().getEventManager().registerListener(new HandshakeListener(this));
        try {
            PacketEvents.getAPI().init();
        } catch (Throwable t) {
            log.error("Failed to initialize packetevents", t);
        }
        EncryptionBypass.install(log);

        getServer().getPluginManager().registerEvents(this, this);
        runVersionCheckAsync();
    }

    public void incrementBypassed() {
        bypassedCount.incrementAndGet();
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
        }.runTaskLater(this, 25L);
    }

    private void sendOpStatus(Player player) {
        player.sendMessage(ChatColor.AQUA + "[OpenFriendBypass] " + ChatColor.RESET + "status:");
        player.sendMessage("  " + ChatColor.GRAY + "key: " +
                (key != null ? ChatColor.GREEN + "loaded" : ChatColor.RED + "missing"));
        player.sendMessage("  " + ChatColor.GRAY + "reflection: " +
                (EncryptionBypass.isReady() ? ChatColor.GREEN + "ready" : ChatColor.YELLOW + "unavailable on this MC version"));
        player.sendMessage("  " + ChatColor.GRAY + "bypassed connections: " + ChatColor.WHITE + bypassedCount.get());
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

    @Override
    public void onDisable() {
        try {
            PacketEvents.getAPI().terminate();
        } catch (Throwable ignored) {
        }
    }

    public BypassKey key() {
        return key;
    }

    public Log log() {
        return log;
    }

    private boolean syncKeyToBridgePlugin(Path keyPath) {
        File pluginsRoot = getDataFolder().getParentFile();
        if (pluginsRoot == null) return false;
        File bridgeDir = new File(pluginsRoot, "OpenFriend");
        if (!bridgeDir.isDirectory()) {
            log.info("OpenFriend bridge plugin folder not found; place bypass.pem manually if running the Go bridge separately.");
            return false;
        }
        Path dest = new File(bridgeDir, "bypass.pem").toPath();
        try {
            Files.copy(keyPath, dest, StandardCopyOption.REPLACE_EXISTING);
            log.info("Copied bypass.pem to " + dest.toAbsolutePath());
            return true;
        } catch (IOException e) {
            log.warn("Failed to copy bypass.pem to OpenFriend bridge dir", e);
            return false;
        }
    }

    private static final class HandshakeListener extends PacketListenerAbstract {
        private final BypassPlugin plugin;

        HandshakeListener(BypassPlugin plugin) {
            this.plugin = plugin;
        }

        @Override
        public void onPacketReceive(PacketReceiveEvent event) {
            if (event.getPacketType() == PacketType.Handshaking.Client.HANDSHAKE) {
                handleHandshake(event);
            } else if (event.getPacketType() == PacketType.Login.Client.LOGIN_START) {
                handleLoginStart(event);
            }
        }

        private void handleHandshake(PacketReceiveEvent event) {
            BypassKey key = plugin.key();
            if (key == null) return;

            WrapperHandshakingClientHandshake handshake = new WrapperHandshakingClientHandshake(event);
            String addr = handshake.getServerAddress();
            BypassKey.MarkerData marker = BypassKey.extract(addr);
            if (marker == null) return;
            if (!key.verify(marker.nonce, marker.signature)) {
                plugin.log().warn("rejected marker with invalid signature from " + event.getUser().getAddress());
                return;
            }
            handshake.setServerAddress(marker.originalAddress);
            event.markForReEncode(true);
            EncryptionBypass.markTrusted((Channel) event.getUser().getChannel());
            plugin.log().info("handshake marker verified; channel marked trusted");
        }

        private void handleLoginStart(PacketReceiveEvent event) {
            Channel channel = (Channel) event.getUser().getChannel();
            if (!EncryptionBypass.isTrusted(channel)) return;
            WrapperLoginClientLoginStart wrapper = new WrapperLoginClientLoginStart(event);
            String name = wrapper.getUsername();
            if (EncryptionBypass.tryBypassLogin(channel, name, plugin.log())) {
                event.setCancelled(true);
                plugin.incrementBypassed();
                plugin.log().info("LoginStart bypassed for " + name);
            }
        }
    }
}
