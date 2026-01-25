# Gson
-keepattributes Signature
-keepattributes *Annotation*
-keep class com.alicia.assistant.model.** { *; }

# Gson TypeToken
-keep class com.google.gson.reflect.TypeToken { *; }
-keep class * extends com.google.gson.reflect.TypeToken

# OkHttp
-dontwarn okhttp3.**
-keep class okhttp3.** { *; }

# Vosk
-keep class org.vosk.** { *; }
-keep class com.sun.jna.** { *; }
-dontwarn com.sun.jna.**

# JNA
-keep class net.java.dev.jna.** { *; }
-dontwarn net.java.dev.jna.**

# OpenTelemetry
-keep class io.opentelemetry.** { *; }
-dontwarn io.opentelemetry.**
-keep class okio.** { *; }
-dontwarn okio.**

# gRPC
-keep class io.grpc.** { *; }
-dontwarn io.grpc.**
