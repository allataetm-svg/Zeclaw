package com.example.frontend

import android.content.Context
import android.content.res.AssetManager
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
                        // return null on success, or log message on failure
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
        val assets = context.resources.assets
        val cacheDir = context.cacheDir
        val backendDir = File(cacheDir, "backend")

        if (!backendDir.exists()) {
            backendDir.mkdirs()
        }

        val abi = android.os.Build.SUPPORTED_ABIS.firstOrNull() ?: "arm64-v8a"
        val binaryName = when {
            abi.contains("arm64") -> "zeclaw-backend-arm64"
            abi.contains("armeabi") -> "zeclaw-backend-arm"
            abi.contains("x86_64") -> "zeclaw-backend-x86_64"
            abi.contains("x86") -> "zeclaw-backend-x86"
            else -> "zeclaw-backend-arm64"
        }

        val binaryFile = File(backendDir, "zeclaw")

        try {
            assets.open("flutter_assets/assets/backend/$binaryName").use { input ->
                binaryFile.outputStream().use { output ->
                    input.copyTo(output)
                }
            }
        } catch (e: Exception) {
            return "Failed to extract backend: ${e.message}"
        }

        if (!binaryFile.exists()) {
            return "Backend binary not found in assets"
        }

        binaryFile.setExecutable(true)
        binaryFile.setReadable(true)

        val candidates = listOf("127.0.0.1:8085", "0.0.0.0:8085", "127.0.0.1:8086")
        val logFile = File(backendDir, "zeclaw.log")
        backendOutput.clear()

        for (addr in candidates) {
            try {
                // Start the backend with ProcessBuilder so we can capture output
                val (host, portStr) = addr.split(":")
                val port = portStr.toInt()
                val pb = ProcessBuilder(binaryFile.absolutePath, "-addr", addr)
                pb.directory(backendDir)
                pb.redirectErrorStream(true)
                // write output to backendDir/zeclaw.log
                pb.redirectOutput(ProcessBuilder.Redirect.appendTo(logFile))
                try {
                    val proc = pb.start()
                    backendProcess = proc
                } catch (e: Exception) {
                    // record and try next candidate
                    backendOutput.append("Failed to exec binary for $addr: ${e.message}\n")
                    continue
                }

                // poll for up to 8s
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

                // also capture a bit of log
                if (logFile.exists()) {
                    val content = logFile.readText()
                    backendOutput.append(content.takeLast(Math.min(content.length, 8000)))
                }

                if (started) {
                    return null // success
                } else {
                    // stop process if still running
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
        // Check if something is listening on host:port
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
}
