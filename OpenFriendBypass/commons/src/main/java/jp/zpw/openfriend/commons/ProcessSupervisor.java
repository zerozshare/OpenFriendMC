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
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.List;
import java.util.concurrent.TimeUnit;

public final class ProcessSupervisor {

    private final List<String> command;
    private final File workingDir;
    private final Log log;
    private final Runnable onUnexpectedExit;

    private Process process;
    private Thread stdoutReader;
    private Thread stderrReader;
    private Thread exitWatcher;
    private volatile boolean shuttingDown;

    public ProcessSupervisor(List<String> command, File workingDir, Log log, Runnable onUnexpectedExit) {
        this.command = command;
        this.workingDir = workingDir;
        this.log = log;
        this.onUnexpectedExit = onUnexpectedExit;
    }

    public boolean start() {
        ProcessBuilder pb = new ProcessBuilder(command).redirectErrorStream(false);
        pb.environment().put("HOME", System.getProperty("user.home", "."));
        if (workingDir != null) {
            pb.directory(workingDir);
        }
        try {
            process = pb.start();
        } catch (IOException e) {
            log.error("Failed to start subprocess", e);
            return false;
        }
        stdoutReader = pipe(process.getInputStream(), false);
        stderrReader = pipe(process.getErrorStream(), true);
        exitWatcher = startExitWatcher();
        log.info("OpenFriend subprocess started: " + String.join(" ", command));
        return true;
    }

    public void stop() {
        shuttingDown = true;
        if (process != null && process.isAlive()) {
            try {
                process.getOutputStream().close();
            } catch (IOException ignored) {
            }
            try {
                if (!process.waitFor(5, TimeUnit.SECONDS)) {
                    log.warn("OpenFriend did not exit in 5s; destroying");
                    process.destroy();
                    if (!process.waitFor(2, TimeUnit.SECONDS)) {
                        process.destroyForcibly();
                    }
                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                process.destroyForcibly();
            }
        }
        joinQuiet(stdoutReader);
        joinQuiet(stderrReader);
        joinQuiet(exitWatcher);
    }

    private Thread pipe(final InputStream in, final boolean asWarn) {
        Thread t = new Thread(new Runnable() {
            public void run() {
                BufferedReader reader = new BufferedReader(new InputStreamReader(in));
                String line;
                try {
                    while ((line = reader.readLine()) != null) {
                        if (asWarn) {
                            log.warn("[openfriend] " + line);
                        } else {
                            log.info("[openfriend] " + line);
                        }
                    }
                } catch (IOException ignored) {
                }
            }
        }, "OpenFriend-pipe");
        t.setDaemon(true);
        t.start();
        return t;
    }

    private Thread startExitWatcher() {
        Thread t = new Thread(new Runnable() {
            public void run() {
                if (process == null) return;
                try {
                    int code = process.waitFor();
                    if (!shuttingDown) {
                        log.warn("OpenFriend subprocess exited unexpectedly (code " + code + ")");
                        if (onUnexpectedExit != null) {
                            try {
                                onUnexpectedExit.run();
                            } catch (RuntimeException ignored) {
                            }
                        }
                    } else {
                        log.info("OpenFriend subprocess exited (code " + code + ")");
                    }
                } catch (InterruptedException ignored) {
                    Thread.currentThread().interrupt();
                }
            }
        }, "OpenFriend-waiter");
        t.setDaemon(true);
        t.start();
        return t;
    }

    private void joinQuiet(Thread t) {
        if (t == null) return;
        try {
            t.join(1000);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }
}
