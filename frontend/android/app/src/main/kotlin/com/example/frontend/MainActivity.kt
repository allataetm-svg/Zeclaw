package com.example.frontend

import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import java.io.File
import java.net.InetSocketAddress
import java.net.Socket
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit

class MainActivity: FlutterActivity() {
    private val CHANNEL = "com.zeclaw.backend/start"

    private var backendProcess: Process? = null
    private val backendOutput = StringBuilder()

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL).setMethodCallHandler { call, result ->
            when (call.method) {
                "startBackend" -> {
                    try {
                        val msg = startBackend()
                        if (msg == null) result.success(true) else result.error("START_ERROR", msg, null)
                    } catch (e: Exception) {
                        result.error("START_ERROR", e.message, null)
                    }
                }
                "isBackendRunning" -> {
                    result.success(isBackendRunning())
                }
                "execShell" -> {
                    val command = call.argument<String>("command") ?: ""
                    val timeout = call.argument<Int>("timeout") ?: 30
                    try {
                        val output = execShell(command, timeout)
                        result.success(output)
                    } catch (e: Exception) {
                        result.error("EXEC_ERROR", e.message, null)
                    }
                }
                "getBackendLog" -> {
                    try {
                        val log = getBackendLog()
                        result.success(log)
                    } catch (e: Exception) {
                        result.error("LOG_ERROR", e.message, null)
                    }
                }
                else -> {
                    result.notImplemented()
                }
            }
        }
    }

    private fun startBackend(): String? {
        val context = applicationContext
        val filesDir = context.filesDir
        val backendDir = File(filesDir, "backend")

        if (!backendDir.exists()) {
            backendDir.mkdirs()
        }

        val binaryFile = File(backendDir, "zeclaw")
        val prootFile = File(backendDir, "proot")

        if (!prootFile.exists()) {
            try {
                val assetManager = context.assets
                val assetFiles = assetManager.list("backend") ?: emptyArray()
                
                val prootAssetName = assetFiles.find { it == "proot" }
                if (prootAssetName != null) {
                    assetManager.open("backend/$prootAssetName").use { input ->
                        prootFile.outputStream().use { output ->
                            input.copyTo(output)
                        }
                    }
                    prootFile.setExecutable(true)
                    prootFile.setReadable(true)
                }
            } catch (e: Exception) {
                return "Failed to extract proot: ${e.message}"
            }
        }

        if (!binaryFile.exists()) {
            try {
                val assetManager = context.assets
                val assetFiles = assetManager.list("backend") ?: emptyArray()
                val binaryAssetName = assetFiles.find { it.contains("zeclaw-backend") }
                if (binaryAssetName == null) {
                    return "Backend binary not found in assets folder"
                }
                assetManager.open("backend/$binaryAssetName").use { input ->
                    binaryFile.outputStream().use { output ->
                        input.copyTo(output)
                    }
                }
                binaryFile.setExecutable(true)
                binaryFile.setReadable(true)
            } catch (e: Exception) {
                return "Failed to extract backend from assets: ${e.message}"
            }
        }

        binaryFile.setExecutable(true)
        binaryFile.setReadable(true)
        prootFile.setExecutable(true)
        prootFile.setReadable(true)

        try {
            Runtime.getRuntime().exec(arrayOf("chmod", "755", prootFile.absolutePath)).waitFor(3, TimeUnit.SECONDS)
            Runtime.getRuntime().exec(arrayOf("chmod", "755", binaryFile.absolutePath)).waitFor(3, TimeUnit.SECONDS)
        } catch (e: Exception) {
            backendOutput.append("chmod failed: ${e.message}\n")
        }

        val candidates = listOf("127.0.0.1:8085", "0.0.0.0:8085", "127.0.0.1:8086")
        val logFile = File(backendDir, "zeclaw.log")
        backendOutput.clear()

        for (addr in candidates) {
            try {
                val (host, portStr) = addr.split(":")
                val port = portStr.toInt()

                val pb = ProcessBuilder()
                pb.command(
                    prootFile.absolutePath,
                    "-0",
                    "-w", backendDir.absolutePath,
                    "-b", "/proc",
                    "-b", "/sys",
                    "-b", "/dev",
                    "-b", "/data/data/com.example.frontend/files:/data/data/com.example.frontend/files",
                    binaryFile.absolutePath,
                    "-addr", addr
                )
                pb.directory(backendDir)
                pb.environment().put("PROOT_TMPDIR", backendDir.absolutePath)
                pb.redirectErrorStream(true)
                pb.redirectOutput(ProcessBuilder.Redirect.appendTo(logFile))

                try {
                    val proc = pb.start()
                    backendProcess = proc
                    backendOutput.append("Started proot with command: ${pb.command().joinToString(" ")}\n")
                } catch (e: Exception) {
                    backendOutput.append("Failed to start proot for $addr: ${e.message}\n")
                    continue
                }

                val start = System.currentTimeMillis()
                val timeoutMs = 8000L
                var started = false
                while (System.currentTimeMillis() - start < timeoutMs) {
                    if (isBackendRunning(host, port)) {
                        started = true
                        break
                    }
                    Thread.sleep(500)
                }

                if (logFile.exists()) {
                    val content = logFile.readText()
                    backendOutput.append(content.takeLast(Math.min(content.length, 8000)))
                }

                if (started) {
                    return null
                } else {
                    try { backendProcess?.destroyForcibly() } catch (_: Exception) {}
                    backendOutput.append("Backend did not start listening on $addr\n")
                }
            } catch (e: Exception) {
                backendOutput.append("Error trying $addr: ${e.message}\n")
            }
        }

        val out = backendOutput.toString()
        return "Backend failed to start on any address. Log:\n$out"
    }

    private fun isBackendRunning(host: String = "127.0.0.1", port: Int = 8085): Boolean {
        try {
            Socket().use { socket ->
                socket.connect(InetSocketAddress(host, port), 500)
                return true
            }
        } catch (e: Exception) {
            return false
        }
    }

    private fun execShell(command: String, timeoutSeconds: Int): String {
        val executor = Executors.newSingleThreadExecutor()
        val filesDir = applicationContext.filesDir
        try {
            val pb = ProcessBuilder("sh", "-c", command)
            pb.directory(filesDir)
            pb.redirectErrorStream(true)
            val process = pb.start()

            val future = executor.submit<String> {
                process.inputStream.bufferedReader().use { it.readText() }
            }

            val output = try {
                future.get(timeoutSeconds.toLong(), TimeUnit.SECONDS)
            } catch (t: Exception) {
                process.destroyForcibly()
                "Command timed out"
            }
            process.waitFor(2, TimeUnit.SECONDS)
            return output
        } finally {
            executor.shutdownNow()
        }
    }

    private fun getBackendLog(): String {
        val filesDir = applicationContext.filesDir
        val logFile = File(filesDir, "backend/zeclaw.log")
        return try {
            if (logFile.exists()) {
                val content = logFile.readText()
                if (content.isEmpty()) "(log empty)" else content
            } else {
                val out = backendOutput.toString()
                if (out.isEmpty()) "(no log file)" else out
            }
        } catch (e: Exception) {
            "Failed to read log: ${e.message}"
        }
    }
}
