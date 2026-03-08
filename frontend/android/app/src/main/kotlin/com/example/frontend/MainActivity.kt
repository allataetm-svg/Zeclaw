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

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL).setMethodCallHandler { call, result ->
            when (call.method) {
                "startBackend" -> {
                    try {
                        startBackend()
                        result.success(true)
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
                else -> {
                    result.notImplemented()
                }
            }
        }
    }

    private fun startBackend() {
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
            throw Exception("Failed to extract backend: ${e.message}")
        }

        if (!binaryFile.exists()) {
            throw Exception("Backend binary not found in assets")
        }

        binaryFile.setExecutable(true)
        binaryFile.setReadable(true)

        // Launch backend in background
        try {
            Runtime.getRuntime().exec(arrayOf("sh", "-c", "chmod 755 ${binaryFile.absolutePath} && ${binaryFile.absolutePath} &"), null, backendDir)
        } catch (e: Exception) {
            throw Exception("Failed to start backend: ${e.message}")
        }
    }

    private fun isBackendRunning(): Boolean {
        // Check if something is listening on localhost:8085
        try {
            Socket().use { socket ->
                socket.connect(InetSocketAddress("127.0.0.1", 8085), 500)
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
