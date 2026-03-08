package com.example.frontend

import android.content.Context
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
        val filesDir = context.filesDir
        val backendDir = File(filesDir, "backend")
        val nativeLibDir = File(context.applicationInfo.nativeLibraryDir)

        if (!backendDir.exists()) {
            backendDir.mkdirs()
        }

        val binaryFile = File(nativeLibDir, "libzeclaw.so")

        // Collect permission/debug info for the binary so we can diagnose execution issues
        val permissionDebugBuilder = StringBuilder()
        try {
            // Ensure executable/readable for owner/group/other
            try {
                binaryFile.setExecutable(true, false)
                binaryFile.setReadable(true, false)
                permissionDebugBuilder.append("setExecutable/setReadable called\n")
            } catch (e: Exception) {
                permissionDebugBuilder.append("Failed to set file flags: ${e.message}\n")
            }

            // Explicit chmod 755 via Runtime.exec
            try {
                val chmodProc = Runtime.getRuntime().exec(arrayOf("chmod", "755", binaryFile.absolutePath))
                val finished = chmodProc.waitFor(3, TimeUnit.SECONDS)
                val stdout = chmodProc.inputStream.bufferedReader().use { it.readText() }
                val stderr = chmodProc.errorStream.bufferedReader().use { it.readText() }
                permissionDebugBuilder.append("chmod finished=${finished} exit=${if (finished) chmodProc.exitValue() else "timeout"}\n")
                if (stdout.isNotBlank()) permissionDebugBuilder.append("chmod stdout:\n${stdout.takeLast(Math.min(stdout.length, 2000))}\n")
                if (stderr.isNotBlank()) permissionDebugBuilder.append("chmod stderr:\n${stderr.takeLast(Math.min(stderr.length, 2000))}\n")
            } catch (e: Exception) {
                permissionDebugBuilder.append("chmod exec failed: ${e.message}\n")
            }

            // ls -l on the binary
            try {
                val lsProc = Runtime.getRuntime().exec(arrayOf("ls", "-l", binaryFile.absolutePath))
                lsProc.waitFor(2, TimeUnit.SECONDS)
                val lsOut = lsProc.inputStream.bufferedReader().use { it.readText().trim() }
                permissionDebugBuilder.append("ls -l ${binaryFile.absolutePath}:\n${lsOut.takeLast(Math.min(lsOut.length, 2000))}\n")
            } catch (e: Exception) {
                permissionDebugBuilder.append("ls -l failed: ${e.message}\n")
            }

            // ls -Z on backendDir to show SELinux context (ignore if not available)
            try {
                val lsZ = Runtime.getRuntime().exec(arrayOf("ls", "-Z", backendDir.absolutePath))
                lsZ.waitFor(2, TimeUnit.SECONDS)
                val out = lsZ.inputStream.bufferedReader().use { it.readText().trim() }
                permissionDebugBuilder.append("ls -Z ${backendDir.absolutePath}:\n${out.takeLast(Math.min(out.length, 2000))}\n")
            } catch (e: Exception) {
                permissionDebugBuilder.append("ls -Z not available or failed: ${e.message}\n")
            }

            // ls -Z on nativeLibDir to show SELinux context of binary location
            try {
                val lsZ = Runtime.getRuntime().exec(arrayOf("ls", "-Z", nativeLibDir.absolutePath))
                lsZ.waitFor(2, TimeUnit.SECONDS)
                val out = lsZ.inputStream.bufferedReader().use { it.readText().trim() }
                permissionDebugBuilder.append("ls -Z ${nativeLibDir.absolutePath}:\n${out.takeLast(Math.min(out.length, 2000))}\n")
            } catch (e: Exception) {
                permissionDebugBuilder.append("ls -Z nativeLibDir failed: ${e.message}\n")
            }

            // /system/bin/id to show uid/gid
            try {
                val idProc = Runtime.getRuntime().exec(arrayOf("/system/bin/id"))
                idProc.waitFor(2, TimeUnit.SECONDS)
                val idOut = idProc.inputStream.bufferedReader().use { it.readText().trim() }
                permissionDebugBuilder.append("/system/bin/id:\n${idOut}\n")
            } catch (e: Exception) {
                permissionDebugBuilder.append("id failed: ${e.message}\n")
            }
        } catch (_: Exception) {
            // keep going; best-effort diagnostics
        }

        val permissionDebug = permissionDebugBuilder.toString()
        if (permissionDebug.isNotEmpty()) {
            backendOutput.append(permissionDebug)
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
                    // re-append permission debug collected earlier so getBackendLog() will include it
                    try {
                        if (permissionDebug.isNotEmpty()) backendOutput.append(permissionDebug)
                    } catch (_: Exception) {}
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

    private fun getBackendLog(): String {
        val filesDir = applicationContext.filesDir
        val logFile = File(filesDir, "backend/zeclaw.log")
        return try {
            if (logFile.exists()) {
                val content = logFile.readText()
                if (content.isEmpty()) "(log empty)" else content
            } else {
                // fallback to captured output
                val out = backendOutput.toString()
                if (out.isEmpty()) "(no log file)" else out
            }
        } catch (e: Exception) {
            "Failed to read log: ${e.message}"
        }
    }
}
