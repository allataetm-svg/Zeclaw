# Zeclaw Development Guide

## Android Build Fix

### Problem
When building the Flutter Android app, the MainActivity.kt fails with errors like:
- `Unresolved reference: io`
- `Unresolved reference: FlutterActivity`
- `Unresolved reference: applicationContext`
- `Unresolved reference: filesDir`

### Root Cause
The Android folder configuration was incompatible with the installed Flutter SDK version (3.24.3). The Flutter embedding library wasn't being resolved properly.

### Solution

1. **Regenerate the Android folder**:
   ```bash
   cd frontend
   mv android android_backup
   flutter create --platforms=android .
   ```

2. **Update Gradle version** to be compatible with Java 17:
   Edit `android/gradle/wrapper/gradle-wrapper.properties`:
   ```properties
   distributionUrl=https\://services.gradle.org/distributions/gradle-8.10.2-all.zip
   ```

3. **Configure Flutter to use Java 17**:
   ```bash
   flutter config --jdk-dir=/usr/lib/jvm/java-17-openjdk-amd64
   ```

4. **Copy custom files back**:
   ```bash
   # Copy MainActivity.kt
   cp android_backup/app/src/main/kotlin/com/example/frontend/MainActivity.kt android/app/src/main/kotlin/com/example/frontend/
   
   # Copy assets folder
   cp -r assets android/app/src/main/
   ```

5. **The key fix**: The MainActivity.kt uses `context.filesDir` instead of `context.cacheDir` for storing the backend binary. This is required because:
   - Android's SELinux policy prevents executing binaries from the cache directory
   - The files directory has proper execute permissions

   In MainActivity.kt, the backend is stored at:
   ```kotlin
   val filesDir = context.filesDir
   val backendDir = File(filesDir, "backend")
   ```

### Building the APK

```bash
cd frontend
flutter build apk --debug
# or for release
flutter build apk --release
```

### Backend Binary
The backend binary for Android ARM64 is located at:
- `frontend/assets/backend/zeclaw-backend-arm64`

Build the backend from the backend folder:
```bash
cd backend
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o ../frontend/assets/backend/zeclaw-backend-arm64 ./cmd/zeclaw/
```
